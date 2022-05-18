package slides

import (
	"context"
	"fmt"
	"log"
	"net"
	"time"

	examplev1 "github.com/hermanbanken/k8s-xds/example/pkg/gen/v1"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/xds"

	clusterservice "github.com/envoyproxy/go-control-plane/envoy/service/cluster/v3"
	discoverygrpc "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	endpointservice "github.com/envoyproxy/go-control-plane/envoy/service/endpoint/v3"
	listenerservice "github.com/envoyproxy/go-control-plane/envoy/service/listener/v3"
	routeservice "github.com/envoyproxy/go-control-plane/envoy/service/route/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"
)

var ctx = context.TODO()

func ExampleNaive() {
	// Connect
	c, err := grpc.DialContext(ctx, "service-b")
	if err != nil {
		log.Fatal(err.Error())
	}
	client := examplev1.NewExampleClient(c)

	// Continuous requests
	for {
		req := &examplev1.ExampleRequest{}
		client.DoSomething(ctx, req)

		time.Sleep(1 * time.Second)
	}
}

func ExampleYAMLHeadless() {
	_ = `
	apiVersion: v1
	kind: Service
	metadata:
	  name: service-b
	spec:
	  type: ClusterIP
	  clusterIP: None
	  selector: { app: service-b }
	  ports:
	  - port: 9090
		protocol: TCP
		targetPort: 9090
	`
}

func ExampleEnvoyYAML() {
	_ = `
admin:
  address:
    socket_address: { address: 127.0.0.1, port_value: 9901 }

static_resources:
  listeners:
  - name: listener_0
    address:
      socket_address: { address: 127.0.0.1, port_value: 80 }
    filter_chains:
    - filters:
      - name: envoy.filters.network.http_connection_manager
        typed_config:
          "@type": type.googleapis.com/envoy.extensions.filters.network.http_connection_manager.v3.HttpConnectionManager
          stat_prefix: ingress_http
          codec_type: AUTO
          route_config:
            name: local_route
            virtual_hosts:
            - name: local_service
              domains: ["*"]
              routes:
              - match: { prefix: "/" }
                route: { cluster: some_service }
          http_filters:
          - name: envoy.filters.http.router
            typed_config:
              "@type": type.googleapis.com/envoy.extensions.filters.http.router.v3.Router
  clusters:
  - name: some_service
    connect_timeout: 0.25s
    type: STRICT_DNS
    lb_policy: ROUND_ROBIN
    load_assignment:
      cluster_name: some_service
      endpoints:
      - lb_endpoints:
        - endpoint:
            address:
              socket_address:
                address: "some_service.default.svc.cluster.local"
                port_value: 50051
`
}

func ExampleXDSClient() {
	c, _ := grpc.DialContext(ctx, "xds:///service-b")
	client := examplev1.NewExampleClient(c)
	client.DoSomething(ctx, &examplev1.ExampleRequest{})
}

var kubernetesWatch func() chan interface{}

func ExampleXDS() {
	snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, nil)
	server := xds.NewServer(ctx, snapshotCache, nil)

	grpcServer := grpc.NewServer()
	discoverygrpc.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	listenerservice.RegisterListenerDiscoveryServiceServer(grpcServer, server)
	routeservice.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	clusterservice.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	endpointservice.RegisterEndpointDiscoveryServiceServer(grpcServer, server)

	lis, _ := net.Listen("tcp", fmt.Sprintf(":%d", 9000))
	go grpcServer.Serve(lis)

	for {
		event := <-kubernetesWatch()
		snapshotCache.SetSnapshot(ctx, "node-id-here", makeSnapshot(event))
	}
}

func makeSnapshot(event interface{}) cache.ResourceSnapshot {
	snapshot, _ := cache.NewSnapshot("1", map[resource.Type][]types.Resource{
		resource.EndpointType: {},
		resource.ClusterType:  {},
		resource.RouteType:    {},
		resource.ListenerType: {},
	})
	return snapshot
}

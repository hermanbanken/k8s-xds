package internal

import (
	"fmt"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"
	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	ep "github.com/envoyproxy/go-control-plane/envoy/api/v2/endpoint"
	listenerv2 "github.com/envoyproxy/go-control-plane/envoy/api/v2/listener"
	v2route "github.com/envoyproxy/go-control-plane/envoy/api/v2/route"
	hcm "github.com/envoyproxy/go-control-plane/envoy/config/filter/network/http_connection_manager/v2"
	lv2 "github.com/envoyproxy/go-control-plane/envoy/config/listener/v2"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/anypb"
	"google.golang.org/protobuf/types/known/wrapperspb"
)

type Mapping = map[string]map[string][]podEndPoint

type podEndPoint struct {
	IP   string
	Port int32
	Zone string
}

// GenerateSnapshot creates snapshot for each service
func GenerateSnapshot(node *core.Node, mapping Mapping) (*cache.Snapshot, error) {
	zap.L().Debug("K8s", zap.Any("EndPoints", mapping))
	var eds []types.Resource
	var cds []types.Resource
	var rds []types.Resource
	var lds []types.Resource
	for service, podEndPoints := range mapping {
		zap.L().Debug("Creating new XDS Entry", zap.String("service", service))
		eds = append(eds, clusterLoadAssignment(podEndPoints, fmt.Sprintf("%s-cluster", service))...)
		cds = append(cds, createCluster(fmt.Sprintf("%s-cluster", service))...)
		rds = append(rds, createRoute(fmt.Sprintf("%s-route", service), fmt.Sprintf("%s-vhost", service), fmt.Sprintf("%s-listener", service), fmt.Sprintf("%s-cluster", service))...)
		lds = append(lds, createListener(fmt.Sprintf("%s-listener", service), fmt.Sprintf("%s-cluster", service), fmt.Sprintf("%s-route", service))...)
	}

	version := uuid.New()
	zap.L().Debug("Creating Snapshot", zap.String("version", version.String()), zap.Any("EDS", eds), zap.Any("CDS", cds), zap.Any("RDS", rds), zap.Any("LDS", lds))
	snapshot := cache.NewSnapshot(version.String(), eds, cds, rds, lds, []types.Resource{}, []types.Resource{})

	if err := snapshot.Consistent(); err != nil {
		zap.L().Error("Snapshot inconsistency", zap.Any("snapshot", snapshot), zap.Error(err))
	}
	return &snapshot, nil
}

func clusterLoadAssignment(zones map[string][]podEndPoint, clusterName string) []types.Resource {
	var lbss = []*ep.LocalityLbEndpoints{}

	for zone, podEndpoints := range zones {
		var lbs []*ep.LbEndpoint
		for _, podEndPoint := range podEndpoints {
			zap.L().Debug("Creating ENDPOINT", zap.String("host", podEndPoint.IP), zap.Int32("port", podEndPoint.Port))
			hst := &core.Address{Address: &core.Address_SocketAddress{
				SocketAddress: &core.SocketAddress{
					Address:  podEndPoint.IP,
					Protocol: core.SocketAddress_TCP,
					PortSpecifier: &core.SocketAddress_PortValue{
						PortValue: uint32(podEndPoint.Port),
					},
				},
			}}

			lbs = append(lbs, &ep.LbEndpoint{
				HostIdentifier: &ep.LbEndpoint_Endpoint{
					Endpoint: &ep.Endpoint{
						Address: hst,
					}},
				HealthStatus: core.HealthStatus_HEALTHY,
			})
		}
		lbss = append(lbss, &ep.LocalityLbEndpoints{
			Locality: &core.Locality{
				Region: zoneToRegion(zone),
				Zone:   zone,
			},
			Priority:            0,
			LoadBalancingWeight: &wrapperspb.UInt32Value{Value: uint32(1000)},
			LbEndpoints:         lbs,
		})
	}

	eds := []types.Resource{
		&v2.ClusterLoadAssignment{
			ClusterName: clusterName,
			Endpoints:   lbss,
		},
	}
	return eds
}

func createCluster(clusterName string) []types.Resource {
	zap.L().Debug("Creating CLUSTER", zap.String("name", clusterName))
	cls := []types.Resource{
		&v2.Cluster{
			Name:                 clusterName,
			LbPolicy:             v2.Cluster_ROUND_ROBIN,
			ClusterDiscoveryType: &v2.Cluster_Type{Type: v2.Cluster_EDS},
			EdsClusterConfig: &v2.Cluster_EdsClusterConfig{
				EdsConfig: &core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_Ads{},
				},
			},
		},
	}
	return cls
}

func createVirtualHost(virtualHostName, listenerName, clusterName string) *v2route.VirtualHost {
	zap.L().Debug("Creating RDS", zap.String("host name", virtualHostName))
	vh := &v2route.VirtualHost{
		Name:    virtualHostName,
		Domains: []string{listenerName},

		Routes: []*v2route.Route{{
			Match: &v2route.RouteMatch{
				PathSpecifier: &v2route.RouteMatch_Prefix{
					Prefix: "",
				},
			},
			Action: &v2route.Route_Route{
				Route: &v2route.RouteAction{
					ClusterSpecifier: &v2route.RouteAction_Cluster{
						Cluster: clusterName,
					},
				},
			},
		}}}
	return vh

}

func createRoute(routeConfigName, virtualHostName, listenerName, clusterName string) []types.Resource {
	vh := createVirtualHost(virtualHostName, listenerName, clusterName)
	rds := []types.Resource{
		&v2.RouteConfiguration{
			Name:         routeConfigName,
			VirtualHosts: []*v2route.VirtualHost{vh},
		},
	}
	return rds
}

func createListener(listenerName string, clusterName string, routeConfigName string) []types.Resource {
	zap.L().Debug("Creating LISTENER", zap.String("name", listenerName))
	hcRds := &hcm.HttpConnectionManager_Rds{
		Rds: &hcm.Rds{
			RouteConfigName: routeConfigName,
			ConfigSource: &core.ConfigSource{
				ConfigSourceSpecifier: &core.ConfigSource_Ads{
					Ads: &core.AggregatedConfigSource{},
				},
			},
		},
	}

	manager := &hcm.HttpConnectionManager{
		CodecType:      hcm.HttpConnectionManager_AUTO,
		RouteSpecifier: hcRds,
	}

	pbst, err := anypb.New(manager)
	if err != nil {
		panic(err)
	}

	lds := []types.Resource{
		&v2.Listener{
			Name: listenerName,
			ApiListener: &lv2.ApiListener{
				ApiListener: pbst,
			},
			Address: &core.Address{
				Address: &core.Address_SocketAddress{
					SocketAddress: &core.SocketAddress{
						Protocol: core.SocketAddress_TCP,
						Address:  "0.0.0.0",
						PortSpecifier: &core.SocketAddress_PortValue{
							PortValue: 10000,
						},
					},
				},
			},
			FilterChains: []*listenerv2.FilterChain{{
				Filters: []*listenerv2.Filter{{
					Name: wellknown.HTTPConnectionManager,
					ConfigType: &listenerv2.Filter_TypedConfig{
						TypedConfig: pbst,
					},
				}},
			}},
		}}
	return lds
}

func zoneToRegion(zone string) string {
	// trim -a -b -c suffix
	return zone[0 : len(zone)-2]
}

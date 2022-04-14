package internal

import (
	"fmt"
	"hash/fnv"
	"math/rand"
	"sort"
	"strings"

	cluster "github.com/envoyproxy/go-control-plane/envoy/config/cluster/v3"
	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	endpoint "github.com/envoyproxy/go-control-plane/envoy/config/endpoint/v3"
	l "github.com/envoyproxy/go-control-plane/envoy/config/listener/v3"
	route "github.com/envoyproxy/go-control-plane/envoy/config/route/v3"
	v3routerpb "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/http/router/v3"
	hcm "github.com/envoyproxy/go-control-plane/envoy/extensions/filters/network/http_connection_manager/v3"
	"github.com/envoyproxy/go-control-plane/pkg/cache/types"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/resource/v3"
	"github.com/envoyproxy/go-control-plane/pkg/wellknown"
	"github.com/google/uuid"
	"go.uber.org/zap"
	"google.golang.org/protobuf/proto"
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
	// Using maximum number of endpoints requires randomness to avoid subsetting the possible large amount of endpoints
	// This requires a seed that is stable per node, so we hash the node id.
	h := fnv.New64a()
	h.Write([]byte(node.Id))
	seed := int64(h.Sum64())

	var ownZone string
	if node.Locality != nil {
		ownZone = node.Locality.Zone
	}

	zap.L().Debug("K8s", zap.Any("EndPoints", mapping))
	var eds []types.Resource
	var cds []types.Resource
	var rds []types.Resource
	var lds []types.Resource
	for service, podEndPoints := range mapping {
		zap.L().Debug("Creating new xDS Entry", zap.String("service", service))
		eds = append(eds, clusterLoadAssignment(podEndPoints, fmt.Sprintf("%s-cluster", service), ownZone, seed)...)
		cds = append(cds, createCluster(fmt.Sprintf("%s-cluster", service))...)
		rds = append(rds, createRoute(fmt.Sprintf("%s-route", service), fmt.Sprintf("%s-vhost", service), service, fmt.Sprintf("%s-cluster", service))...)
		lds = append(lds, createListener(service, fmt.Sprintf("%s-cluster", service), fmt.Sprintf("%s-route", service))...)
	}

	version := uuid.New()
	zap.L().Debug("Creating Snapshot", zap.String("version", version.String()), zap.Any("EDS", eds), zap.Any("CDS", cds), zap.Any("RDS", rds), zap.Any("LDS", lds))
	snapshot, err := cache.NewSnapshot(version.String(), map[resource.Type][]types.Resource{
		resource.EndpointType: eds,
		resource.ClusterType:  cds,
		resource.RouteType:    rds,
		resource.ListenerType: lds,
	})
	if err != nil {
		zap.L().Error("Snapshot error", zap.Any("snapshot", snapshot), zap.Error(err))
	} else if err := snapshot.Consistent(); err != nil {
		zap.L().Error("Snapshot inconsistency", zap.Any("snapshot", snapshot), zap.Error(err))
	}
	return &snapshot, nil
}

func clusterLoadAssignment(zones map[string][]podEndPoint, clusterName string, ownZone string, seed int64) []types.Resource {
	r := rand.New(rand.NewSource(seed))
	cla := &endpoint.ClusterLoadAssignment{ClusterName: clusterName}

	zoneTotal := 0
	zoneNames := []string{}
	for zone, endpoints := range zones {
		zoneNames = append(zoneNames, zone)
		zoneTotal += len(endpoints)
	}

	// Process our own zone first
	prioritySort(zoneNames, ownZone)

	// Add at most max(5, total/3) endpoints to each cluster
	remainingEndpoints := zoneTotal / 3
	if remainingEndpoints < 5 {
		remainingEndpoints = 5
	}

outerLoop:
	for _, zone := range zoneNames {
		podEndpoints := zones[zone]

		// Locality Weighted Load Balancing
		// @see https://www.envoyproxy.io/docs/envoy/latest/intro/arch_overview/upstream/load_balancing/locality_weight
		var weight uint32 = 1
		if zone == ownZone {
			weight = 1000
		}
		var locality = &endpoint.LocalityLbEndpoints{
			Locality: &core.Locality{
				Region: zoneToRegion(zone),
				Zone:   zone,
			},
			Priority:            0,
			LoadBalancingWeight: &wrapperspb.UInt32Value{Value: weight},
		}
		cla.Endpoints = append(cla.Endpoints, locality)

		sort.Slice(podEndpoints, func(i, j int) bool {
			return strings.Compare(podEndpoints[i].IP, podEndpoints[j].IP) < 0
		})
		randomForEach(podEndpoints, r, func(i int) {
			if remainingEndpoints == 0 {
				return
			}

			podEndPoint := podEndpoints[i]
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
			locality.LbEndpoints = append(locality.LbEndpoints, &endpoint.LbEndpoint{
				HostIdentifier: &endpoint.LbEndpoint_Endpoint{
					Endpoint: &endpoint.Endpoint{
						Address: hst,
					}},
				HealthStatus: core.HealthStatus_HEALTHY,
			})
			remainingEndpoints--
		})
		if remainingEndpoints == 0 {
			break outerLoop
		}

	}

	return []types.Resource{cla}
}

func createCluster(clusterName string) []types.Resource {
	zap.L().Debug("Creating CLUSTER", zap.String("name", clusterName))
	cls := []types.Resource{
		&cluster.Cluster{
			Name:                 clusterName,
			LbPolicy:             cluster.Cluster_ROUND_ROBIN,
			ClusterDiscoveryType: &cluster.Cluster_Type{Type: cluster.Cluster_EDS},
			EdsClusterConfig: &cluster.Cluster_EdsClusterConfig{
				EdsConfig: &core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_Ads{},
				},
			},
		},
	}
	return cls
}

func createVirtualHost(virtualHostName, listenerName, clusterName string) *route.VirtualHost {
	zap.L().Debug("Creating RDS", zap.String("host name", virtualHostName))
	vh := &route.VirtualHost{
		Name:    virtualHostName,
		Domains: []string{listenerName},

		Routes: []*route.Route{{
			Match: &route.RouteMatch{
				PathSpecifier: &route.RouteMatch_Prefix{
					Prefix: "",
				},
			},
			Action: &route.Route_Route{
				Route: &route.RouteAction{
					ClusterSpecifier: &route.RouteAction_Cluster{
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
		&route.RouteConfiguration{
			Name:         routeConfigName,
			VirtualHosts: []*route.VirtualHost{vh},
		},
	}
	return rds
}

func createListener(listenerName string, clusterName string, routeConfigName string) []types.Resource {
	zap.L().Debug("Creating LISTENER", zap.String("name", listenerName))
	pbst := any(&hcm.HttpConnectionManager{
		CodecType: hcm.HttpConnectionManager_AUTO,
		RouteSpecifier: &hcm.HttpConnectionManager_Rds{
			Rds: &hcm.Rds{
				RouteConfigName: routeConfigName,
				ConfigSource: &core.ConfigSource{
					ConfigSourceSpecifier: &core.ConfigSource_Ads{
						Ads: &core.AggregatedConfigSource{},
					},
				},
			},
		},
		HttpFilters: []*hcm.HttpFilter{{
			Name: "router",
			ConfigType: &hcm.HttpFilter_TypedConfig{
				TypedConfig: any(&v3routerpb.Router{}),
			},
		}},
	})
	lds := []types.Resource{
		&l.Listener{
			Name: listenerName,
			ApiListener: &l.ApiListener{
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
			FilterChains: []*l.FilterChain{{
				Filters: []*l.Filter{{
					Name: wellknown.HTTPConnectionManager,
					ConfigType: &l.Filter_TypedConfig{
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

func any(m proto.Message) *anypb.Any {
	a, err := anypb.New(m)
	if err != nil {
		panic(err)
	}
	return a
}

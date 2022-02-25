package internal

import (
	"context"
	"fmt"
	"net"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"

	xds "github.com/envoyproxy/go-control-plane/pkg/server/v2"

	"go.uber.org/zap"
	"google.golang.org/grpc"

	discovery "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v2"
)

// const grpcMaxConcurrentStreams = 1000

func registerServices(grpcServer *grpc.Server, server xds.Server) {
	discovery.RegisterAggregatedDiscoveryServiceServer(grpcServer, server)
	v2.RegisterEndpointDiscoveryServiceServer(grpcServer, server)
	v2.RegisterClusterDiscoveryServiceServer(grpcServer, server)
	v2.RegisterRouteDiscoveryServiceServer(grpcServer, server)
	v2.RegisterListenerDiscoveryServiceServer(grpcServer, server)
}

// RunManagementServer starts an xDS server at the given port.
func RunManagementServer(ctx context.Context, server xds.Server, port uint, maxConcurrentStreams uint32) {
	var grpcOptions []grpc.ServerOption
	grpcOptions = append(grpcOptions, grpc.MaxConcurrentStreams(maxConcurrentStreams))
	grpcServer := grpc.NewServer(grpcOptions...)

	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		zap.L().Error("Failed to listen", zap.Error(err))
	}

	// register services
	registerServices(grpcServer, server)

	zap.L().Info("Management server listening", zap.Uint("port", port))
	go func() {
		if err = grpcServer.Serve(lis); err != nil {
			zap.L().Error("Failed to start management server", zap.Error(err))
		}
	}()
	<-ctx.Done()

	grpcServer.GracefulStop()
}
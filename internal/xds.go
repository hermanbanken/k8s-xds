package internal

import (
	"context"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func Run(ctx context.Context, config *viper.Viper, d Discovery) {
	upstreamServices := config.GetStringSlice("upstreamServices")

	signal := make(chan struct{})
	cb := &Callbacks{
		Signal:   signal,
		Fetches:  0,
		Requests: 0,
	}

	go func() {
		err := d.Start(ctx, upstreamServices)
		if err != nil {
			zap.L().Fatal("discovery crashed", zap.Error(err))
		}
	}()

	filterCache := &FilterCache{
		createFn: func(node *core.Node) cache.SnapshotCache {
			zap.L().Info("Creating Node", zap.String("Id", node.Id))
			// ads=false to disable ADS: otherwise the xDS server will wait with responding until the
			// xDS client lists all resource names (which it never will if it just utilizes a subset)
			// link: https://github.com/grpc/grpc-go/issues/5131#issuecomment-1022434793
			snapshotCache := cache.NewSnapshotCache(false, cache.IDHash{}, xdsLog())
			stream := d.Watch()
			go func() {
				for {
					m := <-stream
					zap.L().Debug("New mapping", zap.Any("mapping", m))
					ss, err := GenerateSnapshot(node, m)
					if err != nil {
						zap.L().Error("Error in Generating the SnapShot", zap.Error(err))
						return
					}
					snapshotCache.SetSnapshot(ctx, node.Id, ss)
				}
			}()
			return snapshotCache
		},
	}

	srv := xds.NewServer(ctx, filterCache, cb)
	RunManagementServer(ctx, srv, uint(config.GetInt("managementServer.port")), uint32(config.GetInt("maxConcurrentStreams")))
}

func Contains(sl []string, str string) bool {
	for _, s := range sl {
		if s == str {
			return true
		}
	}
	return false
}

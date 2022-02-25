package internal

import (
	"context"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v2"
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

	go d.Start(ctx, upstreamServices)

	filterCache := &FilterCache{
		createFn: func(node *core.Node) cache.SnapshotCache {
			zap.L().Info("Creating Node", zap.String("Id", node.Id))
			snapshotCache := cache.NewSnapshotCache(true, cache.IDHash{}, xdsLog())
			stream := d.Watch()
			go func() {
				for {
					m := <-stream
					ss, err := GenerateSnapshot(node, m)
					if err != nil {
						zap.L().Error("Error in Generating the SnapShot", zap.Error(err))
						return
					}
					snapshotCache.SetSnapshot(node.Id, *ss)
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

package internal

import (
	"context"
	"os"

	core "github.com/envoyproxy/go-control-plane/envoy/api/v2/core"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

func Run(ctx context.Context) {
	config, err := ReadConfig()
	if err != nil {
		zap.L().Fatal(err.Error())
	}
	upstreamServices := config.GetStringSlice("upstreamServices")

	signal := make(chan struct{})
	cb := &Callbacks{
		Signal:   signal,
		Fetches:  0,
		Requests: 0,
	}

	d := Discovery{}
	go d.Start(ctx, upstreamServices)

	filterCache := &FilterCache{
		createFn: func(node *core.Node) cache.SnapshotCache {
			zap.L().Info("Creating Node", zap.String("Id", node.Id))
			snapshotCache := cache.NewSnapshotCache(true, cache.IDHash{}, xdsLog())
			go func() {
				for {
					m := <-d.Watch()
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

// ReadConfig reads the config data from file
func ReadConfig() (*viper.Viper, error) {
	file := "/var/run/config/app.yaml"
	if fileEnv, hasEnv := os.LookupEnv("CONFIG"); hasEnv {
		file = fileEnv
	}
	zap.L().Debug("Reading configuration", zap.String("file", file))
	v := viper.New()
	v.SetConfigFile(file)
	v.AutomaticEnv()
	err := v.ReadInConfig()
	return v, err
}

func Contains(sl []string, str string) bool {
	for _, s := range sl {
		if s == str {
			return true
		}
	}
	return false
}

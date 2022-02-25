package internal

import (
	"context"
	"encoding/json"
	"log"
	"os"
	"time"

	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v2"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v2"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/watch"
)

func computeMapping(data map[string]Slice) {
	mapping := map[string]map[string][]string{}
	for _, slice := range data {
		if _, hasService := mapping[slice.Service]; !hasService {
			mapping[slice.Service] = map[string][]string{}
		}
		for _, e := range slice.Endpoints {
			mapping[slice.Service][e.Topology.Zone] = append(mapping[slice.Service][e.Topology.Zone], e.Addresses...)
		}
	}
	b, _ := json.MarshalIndent(mapping, "", "  ")
	log.Println(string(b))
}

func Run(ctx context.Context) {
	config, err := ReadConfig()
	if err != nil {
		zap.L().Fatal(err.Error())
	}
	upstreamServices := config.GetStringSlice("upstreamServices")
	slices := make(map[string]Slice)

	go dowatch(ctx, func(t watch.EventType, s Slice) {
		if len(upstreamServices) > 0 && !Contains(upstreamServices, s.Service) {
			return
		}
		if t == watch.Added || t == watch.Modified {
			slices[s.Name] = s
		} else if t == watch.Deleted {
			delete(slices, s.Name)
		}
		computeMapping(slices)
	})

	signal := make(chan struct{})
	cb := &Callbacks{
		Signal:   signal,
		Fetches:  0,
		Requests: 0,
	}

	snapshotCache := cache.NewSnapshotCache(true, cache.IDHash{}, xdsLog())
	srv := xds.NewServer(ctx, snapshotCache, cb)
	go RunManagementServer(ctx, srv, uint(config.GetInt("managementServer.port")), uint32(config.GetInt("maxConcurrentStreams")))
	<-signal

	cb.Report()

	zap.L().Debug("Status", zap.Any("keys", snapshotCache.GetStatusKeys()))

	nodeID := config.GetString("nodeId")
	zap.L().Info("Creating Node", zap.String("Id", nodeID))
	for {
		ss, err := GenerateSnapshot(config.GetStringSlice("upstreamServices"))
		if err != nil {
			zap.L().Error("Error in Generating the SnapShot", zap.Error(err))
		} else {
			snapshotCache.SetSnapshot(nodeID, *ss)
			time.Sleep(60 * time.Second)
		}
	}
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

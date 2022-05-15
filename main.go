package main

import (
	"context"
	"os"

	"github.com/hermanbanken/k8s-xds/internal"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var levelFlag = zap.LevelFlag("loglevel", zap.DebugLevel, "set the loglevel")

func main() {
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(zap.NewProductionEncoderConfig()),
		os.Stderr,
		levelFlag,
	)
	z := zap.New(core)
	zap.ReplaceGlobals(z)
	zap.L().Info("Starting control plane")
	ctx := context.Background()
	config, err := ReadConfig()
	if err != nil {
		zap.L().Fatal(err.Error())
	}

	internal.Run(ctx, config, &internal.DiscoveryImpl{Fn: internal.KubernetesEndpointWatch})
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
	if err != nil {
		zap.L().Error("read config error", zap.Error(err))
	} else {
		zap.L().Debug("read config", zap.Any("config", v.AllSettings()))
	}
	return v, err
}

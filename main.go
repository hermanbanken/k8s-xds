package main

import (
	"context"

	"github.com/hermanbanken/k8s-xds/internal"
	"go.uber.org/zap"
)

func main() {
	z, _ := zap.NewProduction()
	zap.ReplaceGlobals(z)
	zap.L().Info("Starting control plane")
	ctx := context.Background()
	internal.Run(ctx)
}

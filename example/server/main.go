package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"

	examplev1 "github.com/hermanbanken/k8s-xds/example/pkg/gen/v1"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"google.golang.org/grpc"
)

var cleanupTracing = installExportPipeline(context.Background(), "server")

func main() {
	grpcServer := grpc.NewServer(
		grpc.UnaryInterceptor(otelgrpc.UnaryServerInterceptor()),
		grpc.StreamInterceptor(otelgrpc.StreamServerInterceptor()),
	)
	examplev1.RegisterExampleServer(grpcServer, example{})
	zap.L().Info("Listening on :9090")
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal(err)
	}

	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt)

	go grpcServer.Serve(lis)

	// Stop on signal
	sig := <-c
	zap.S().Info("Got %s signal. Aborting...\n", sig)
	grpcServer.GracefulStop()
	cleanupTracing()
}

type example struct {
	examplev1.UnimplementedExampleServer
}

func (example) DoSomething(ctx context.Context, req *examplev1.ExampleRequest) (*examplev1.ExampleResponse, error) {
	return &examplev1.ExampleResponse{Message: fmt.Sprintf("Hi %s", req.Name)}, nil
}

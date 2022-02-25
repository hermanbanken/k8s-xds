package main

import (
	"context"
	"fmt"
	"log"
	"net"

	examplev1 "github.com/hermanbanken/k8s-xds/example/pkg/gen/v1"
	"google.golang.org/grpc"
)

func main() {
	grpcServer := grpc.NewServer()
	examplev1.RegisterExampleServer(grpcServer, example{})
	log.Println("Listening on :9090")
	lis, err := net.Listen("tcp", ":9090")
	if err != nil {
		log.Fatal(err)
	}
	grpcServer.Serve(lis)
}

type example struct {
	examplev1.UnimplementedExampleServer
}

func (example) DoSomething(ctx context.Context, req *examplev1.ExampleRequest) (*examplev1.ExampleResponse, error) {
	return &examplev1.ExampleResponse{Message: fmt.Sprintf("Hi %s", req.Name)}, nil
}

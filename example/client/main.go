package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	examplev1 "github.com/hermanbanken/k8s-xds/example/pkg/gen/v1"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
)

func main() {
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	stop := make(chan os.Signal)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-stop
		cancel()
	}()

	c, err := grpc.DialContext(ctx, os.Getenv("upstream_host"), grpc.WithInsecure())
	if err != nil {
		log.Fatal(err.Error())
	}

	resp, err := examplev1.NewExampleClient(c).DoSomething(ctx, &examplev1.ExampleRequest{
		Name: "hello world",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(resp.Message)
}

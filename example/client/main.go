package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

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

	c, err := grpc.DialContext(ctx, os.Getenv("upstream_host"))
	if err != nil {
		log.Panic(err.Error())
	}

	c.Invoke(ctx, "DoSomething", struct{}{}, struct{}{})
}

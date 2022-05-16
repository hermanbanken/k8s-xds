package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	examplev1 "github.com/hermanbanken/k8s-xds/example/pkg/gen/v1"
	etrace "github.com/hermanbanken/k8s-xds/example/trace"
	"github.com/jnovack/flag"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/xds"
)

var rate = flag.Duration("duration", time.Second, "how often to emit a message")
var host = flag.String("upstream_host", "", "grpc destination server uri")

var cleanupTracing = etrace.InstallExportPipeline(context.Background(), "client")

func main() {
	defer cleanupTracing()
	flag.Parse()

	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()

	stop := make(chan os.Signal)
	signal.Notify(stop, syscall.SIGTERM, syscall.SIGINT)

	go func() {
		<-stop
		cancel()
	}()

	c, err := grpc.DialContext(ctx, *host,
		grpc.WithInsecure(),
		grpc.WithUnaryInterceptor(otelgrpc.UnaryClientInterceptor()))
	if err != nil {
		log.Fatal(err.Error())
	}

	selfIp := etrace.GetLocalIP()
	t := time.NewTicker(*rate)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			resp, err := examplev1.NewExampleClient(c).DoSomething(ctx, &examplev1.ExampleRequest{
				Name: selfIp,
			})
			if err != nil {
				log.Fatal(err)
			}
			log.Println("response", resp.Message)
		}
	}
}

package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	examplev1 "github.com/hermanbanken/k8s-xds/example/pkg/gen/v1"
	etrace "github.com/hermanbanken/k8s-xds/example/trace"
	"github.com/jnovack/flag"
	"go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc"
	"go.uber.org/zap"
	"golang.org/x/net/context"
	"google.golang.org/grpc"
	_ "google.golang.org/grpc/xds"
)

var rate = flag.Duration("duration", time.Second, "how often to emit a message")
var host = flag.String("upstream_host", "", "grpc destination server uri")

var cleanupTracing = etrace.InstallExportPipeline(context.Background(), "client")

func main() {
	defer cleanupTracing()
	etrace.InstallZap()
	flag.Parse()
	selfIp := etrace.GetLocalIP()
	zap.L().Info(selfIp)

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
	client := examplev1.NewExampleClient(c)

	// Demo Kubernetes health endpoint
	server := &http.Server{Addr: ":8080", Handler: http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		_, err := client.DoSomething(ctx, &examplev1.ExampleRequest{
			Name: selfIp,
		})
		if err != nil {
			zap.L().Warn("Health check failed")
			rw.WriteHeader(500)
			rw.Write([]byte(err.Error()))
			return
		}
		zap.L().Debug("Health check ok")
		rw.WriteHeader(200)
	})}
	defer server.Close()
	go func() {
		err := server.ListenAndServe()
		if err != nil {
			log.Fatal(err.Error())
		}
	}()

	// Some busy work
	t := time.NewTicker(*rate)
	for {
		select {
		case <-ctx.Done():
			return
		case <-t.C:
			resp, err := client.DoSomething(ctx, &examplev1.ExampleRequest{
				Name: selfIp,
			})
			if err == nil {
				log.Println("response", resp.Message)
			} else {
				zap.L().Warn("failure", zap.Error(err))
			}
		}
	}
}

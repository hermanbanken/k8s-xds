package internal

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"testing"
	"time"

	examplev1 "github.com/hermanbanken/k8s-xds/example/pkg/gen/v1"
	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	"google.golang.org/grpc"
	"google.golang.org/grpc/xds"
)

func TestXds(t *testing.T) {
	z, _ := zap.NewDevelopment()
	zap.ReplaceGlobals(z)

	t.Log("starting servers")
	// Example Servers
	go runServer(8000)
	go runServer(8001)
	go runServer(8002)

	// XDS
	t.Logf("Starting control plane on :9000")
	ctx, cancel := context.WithCancel(context.TODO())
	defer cancel()
	config := viper.New()
	config.Set("nodeId", "foo")
	config.Set("maxConcurrentStreams", 1000)
	config.Set("managementServer.port", 9000)
	config.Set("upstreamServices", []string{"example-server"})
	discovery := &MockDiscovery{filePath: "!manual!"}
	go Run(ctx, config, discovery)
	time.Sleep(time.Second)

	// Configuration A
	discovery.Emit(map[string]map[string][]podEndPoint{
		"example-server": {
			"europe-west4-a": {{IP: "127.0.0.1", Port: 8000, Zone: "europe-west4-a"}},
			"europe-west4-b": {{IP: "127.0.0.1", Port: 8001, Zone: "europe-west4-b"}},
			"europe-west4-c": {{IP: "127.0.0.1", Port: 8002, Zone: "europe-west4-c"}},
		},
	})

	t.Logf("Starting client")
	bootstrap, err := os.ReadFile("../example/bootstrap.json")
	if !assert.NoError(t, err) {
		return
	}
	t.Log(string(bootstrap))
	resolver, err := xds.NewXDSResolverWithConfigForTesting(bootstrap)
	assert.NoError(t, err)
	c, err := grpc.DialContext(ctx, "xds:///example-server-listener", grpc.WithInsecure(), grpc.WithResolvers(resolver))
	if err != nil {
		log.Fatal(err.Error())
	}

	assert.Equal(t, "Hi hello world from 8000", runClient(ctx, c))

	discovery.Emit(map[string]map[string][]podEndPoint{
		"example-server": {
			"europe-west4-b": {{IP: "127.0.0.1", Port: 8001, Zone: "europe-west4-b"}},
		},
	})
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "Hi hello world from 8001", runClient(ctx, c))

	discovery.Emit(map[string]map[string][]podEndPoint{
		"example-server": {
			"europe-west4-c": {{IP: "127.0.0.1", Port: 8002, Zone: "europe-west4-c"}},
		},
	})
	time.Sleep(100 * time.Millisecond)
	assert.Equal(t, "Hi hello world from 8002", runClient(ctx, c))

}

func runServer(port int) {
	grpcServer := grpc.NewServer()
	examplev1.RegisterExampleServer(grpcServer, example{port: port})
	log.Printf("Listening on :%d\n", port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
	if err != nil {
		log.Fatal(err)
	}
	grpcServer.Serve(lis)
}

type example struct {
	examplev1.UnimplementedExampleServer
	port int
}

func (e example) DoSomething(ctx context.Context, req *examplev1.ExampleRequest) (*examplev1.ExampleResponse, error) {
	return &examplev1.ExampleResponse{Message: fmt.Sprintf("Hi %s from %d", req.Name, e.port)}, nil
}

func runClient(ctx context.Context, c grpc.ClientConnInterface) string {
	resp, err := examplev1.NewExampleClient(c).DoSomething(ctx, &examplev1.ExampleRequest{
		Name: "hello world",
	})
	if err != nil {
		log.Fatal(err)
	}
	return resp.Message
}

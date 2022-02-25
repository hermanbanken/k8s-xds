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
	"google.golang.org/grpc"
	"google.golang.org/grpc/xds"
)

func TestXds(t *testing.T) {
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
	go Run(ctx, config, &MockDiscovery{filePath: "mapping.yaml"})

	time.Sleep(time.Second)

	t.Logf("Starting client")
	bootstrap, err := os.ReadFile("../example/bootstrap.json")
	if !assert.NoError(t, err) {
		return
	}
	t.Log(string(bootstrap))
	resolver, err := xds.NewXDSResolverWithConfigForTesting(bootstrap)
	assert.NoError(t, err)
	c, err := grpc.DialContext(ctx, "xds:///example-server", grpc.WithInsecure(), grpc.WithResolvers(resolver))
	if err != nil {
		log.Fatal(err.Error())
	}
	runClient(ctx, c)

	// go func() {
	// 	// test
	// 	node := &core.Node{Id: "foobar", Locality: &core.Locality{Zone: ""}}
	// 	snapshotCache := cache.NewSnapshotCache(true, cache.IDHash{}, xdsLog())
	// 	stream := d.Watch()
	// 	for {
	// 		m := <-stream
	// 		ss, err := GenerateSnapshot(node, m)
	// 		if err != nil {
	// 			zap.L().Error("Error in Generating the SnapShot", zap.Error(err))
	// 			return
	// 		}
	// 		snapshotCache.SetSnapshot(ctx, node.Id, *ss)
	// 		zap.S().Info(ss)
	// 	}
	// }()
}

func runServer(port int) {
	grpcServer := grpc.NewServer()
	examplev1.RegisterExampleServer(grpcServer, example{})
	log.Printf("Listening on :%d\n", port)
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", port))
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

func runClient(ctx context.Context, c grpc.ClientConnInterface) {
	resp, err := examplev1.NewExampleClient(c).DoSomething(ctx, &examplev1.ExampleRequest{
		Name: "hello world",
	})
	if err != nil {
		log.Fatal(err)
	}
	log.Println(resp.Message)
}

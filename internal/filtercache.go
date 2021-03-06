package internal

import (
	"context"
	"crypto/sha256"
	"fmt"
	"sync"

	core "github.com/envoyproxy/go-control-plane/envoy/config/core/v3"
	cache "github.com/envoyproxy/go-control-plane/pkg/cache/v3"
	"github.com/envoyproxy/go-control-plane/pkg/server/stream/v3"
)

type FilterCache struct {
	sync.Mutex
	lookup map[string]struct {
		cache.SnapshotCache
		*core.Node
	}
	createFn func(node *core.Node) cache.SnapshotCache
}

var _ cache.Cache = &FilterCache{}

func (fc *FilterCache) get(node *core.Node) cache.Cache {
	fc.Lock()
	defer fc.Unlock()
	key := AsSha256(node)
	if fc.lookup == nil {
		fc.lookup = make(map[string]struct {
			cache.SnapshotCache
			*core.Node
		})
	}
	if _, has := fc.lookup[key]; !has {
		fc.lookup[key] = struct {
			cache.SnapshotCache
			*core.Node
		}{
			SnapshotCache: fc.createFn(node),
			Node:          node,
		}
	}
	return fc.lookup[key].SnapshotCache
}

var _ cache.Cache = &FilterCache{}

func (fc *FilterCache) CreateWatch(req *cache.Request, ss stream.StreamState, resp chan cache.Response) (cancel func()) {
	return fc.get(req.Node).CreateWatch(req, ss, resp)
}

func (fc *FilterCache) CreateDeltaWatch(req *cache.DeltaRequest, ss stream.StreamState, resp chan cache.DeltaResponse) (cancel func()) {
	return fc.get(req.Node).CreateDeltaWatch(req, ss, resp)
}

func (fc *FilterCache) Fetch(ctx context.Context, req *cache.Request) (cache.Response, error) {
	return fc.get(req.Node).Fetch(ctx, req)
}

// https://blog.8bitzen.com/posts/22-08-2019-how-to-hash-a-struct-in-go
func AsSha256(o interface{}) string {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("%v", o)))

	return fmt.Sprintf("%x", h.Sum(nil))
}

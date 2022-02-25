package internal

import (
	"context"
	"sync"

	v2 "github.com/envoyproxy/go-control-plane/envoy/api/v2"

	"go.uber.org/zap"
)

// Report type
func (cb *Callbacks) Report() {
	cb.mu.Lock()
	defer cb.mu.Unlock()
	zap.L().Debug("cb.Report()  callbacks", zap.Any("Fetches", cb.Fetches), zap.Any("Requests", cb.Requests))
}

// OnStreamOpen type
func (cb *Callbacks) OnStreamOpen(ctx context.Context, id int64, typ string) error {
	zap.L().Debug("OnStreamOpen", zap.Int64("id", id), zap.String("type", typ))
	return nil
}

// OnStreamClosed type
func (cb *Callbacks) OnStreamClosed(id int64) {
	zap.L().Debug("OnStreamClosed", zap.Int64("id", id))
}

// OnStreamRequest type
func (cb *Callbacks) OnStreamRequest(id int64, req *v2.DiscoveryRequest) error {
	zap.L().Debug("OnStreamRequest", zap.Int64("id", id), zap.Any("Request", req))
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Requests++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}
	return nil
}

// OnStreamResponse type
func (cb *Callbacks) OnStreamResponse(id int64, req *v2.DiscoveryRequest, resp *v2.DiscoveryResponse) {
	zap.L().Debug("OnStreamResponse", zap.Int64("id", id), zap.Any("Request", req), zap.Any("Response ", resp))
	cb.Report()
}

// OnFetchRequest type
func (cb *Callbacks) OnFetchRequest(ctx context.Context, req *v2.DiscoveryRequest) error {
	zap.L().Debug("OnFetchRequest", zap.Any("Request", req))
	cb.mu.Lock()
	defer cb.mu.Unlock()
	cb.Fetches++
	if cb.Signal != nil {
		close(cb.Signal)
		cb.Signal = nil
	}
	return nil
}

// OnFetchResponse type
func (cb *Callbacks) OnFetchResponse(req *v2.DiscoveryRequest, resp *v2.DiscoveryResponse) {
	zap.L().Debug("OnFetchResponse", zap.Any("Request", req), zap.Any("Response", resp))
}

// Callbacks for XD Server
type Callbacks struct {
	Signal   chan struct{}
	Fetches  int
	Requests int
	mu       sync.Mutex
}

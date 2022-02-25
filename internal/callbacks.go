package internal

import (
	"context"
	"sync"

	discoveryv3 "github.com/envoyproxy/go-control-plane/envoy/service/discovery/v3"
	xds "github.com/envoyproxy/go-control-plane/pkg/server/v3"

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

// OnDeltaStreamOpen is called once an incremental xDS stream is open with a stream ID and the type URL (or "" for ADS).
// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
func (cb *Callbacks) OnDeltaStreamOpen(ctx context.Context, id int64, typ string) error {
	zap.L().Debug("", zap.Int64("id", id), zap.String("type", typ))
	return nil
}

// OnDeltaStreamClosed is called immediately prior to closing an xDS stream with a stream ID.
func (cb *Callbacks) OnDeltaStreamClosed(id int64) {
	zap.L().Debug("", zap.Int64("id", id))
}

// OnStreamDeltaRequest is called once a request is received on a stream.
// Returning an error will end processing and close the stream. OnStreamClosed will still be called.
func (cb *Callbacks) OnStreamDeltaRequest(id int64, req *discoveryv3.DeltaDiscoveryRequest) error {
	zap.L().Debug("", zap.Int64("id", id))
	return nil
}

// OnStreamDelatResponse is called immediately prior to sending a response on a stream.
func (cb *Callbacks) OnStreamDeltaResponse(id int64, req *discoveryv3.DeltaDiscoveryRequest, resp *discoveryv3.DeltaDiscoveryResponse) {
	zap.L().Debug("", zap.Int64("id", id))
}

// OnStreamRequest type
func (cb *Callbacks) OnStreamRequest(id int64, req *discoveryv3.DiscoveryRequest) error {
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
func (cb *Callbacks) OnStreamResponse(ctx context.Context, id int64, req *discoveryv3.DiscoveryRequest, resp *discoveryv3.DiscoveryResponse) {
	zap.L().Debug("OnStreamResponse", zap.Int64("id", id), zap.Any("Request", req), zap.Any("Response ", resp))
	cb.Report()
}

// OnFetchRequest type
func (cb *Callbacks) OnFetchRequest(ctx context.Context, req *discoveryv3.DiscoveryRequest) error {
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
func (cb *Callbacks) OnFetchResponse(req *discoveryv3.DiscoveryRequest, resp *discoveryv3.DiscoveryResponse) {
	zap.L().Debug("OnFetchResponse", zap.Any("Request", req), zap.Any("Response", resp))
}

// Callbacks for XD Server
type Callbacks struct {
	Signal   chan struct{}
	Fetches  int
	Requests int
	mu       sync.Mutex
}

var _ xds.Callbacks = &Callbacks{}

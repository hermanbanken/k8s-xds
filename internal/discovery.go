package internal

import (
	"context"
	"io/ioutil"
	"os"
	"sync"
	"time"

	"github.com/bep/debounce"
	"go.uber.org/zap"
	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/watch"
)

type Discovery interface {
	Start(ctx context.Context, upstreamServices []string) error
	Watch() <-chan Mapping
}

// DiscoveryImpl is a generic discovery layer that hooks to Fn.
// It generates and emits zoned mappings, by inspecting the Slice's Endpoint information.
type DiscoveryImpl struct {
	sync.Mutex
	last    Mapping
	workers []func(Mapping)
	Fn      func(context.Context, func(t watch.EventType, s Slice)) error
}

func (d *DiscoveryImpl) Start(ctx context.Context, upstreamServices []string) error {
	slices := make(map[string]Slice)
	debounced := debounce.New(50 * time.Millisecond)
	return d.Fn(ctx, func(t watch.EventType, s Slice) {
		if len(upstreamServices) > 0 && !Contains(upstreamServices, s.Service) {
			return
		}
		if t == watch.Added || t == watch.Modified {
			slices[s.Name] = s
		} else if t == watch.Deleted {
			delete(slices, s.Name)
		}
		debounced(func() {
			m := d.computeMapping(slices)
			d.Emit(m)
		})
	})
}

// computeMapping converts from EndpointSlices to a zoned mapping so the downstream services do not need to transform individually
func (d *DiscoveryImpl) computeMapping(slices map[string]Slice) Mapping {
	mapping := Mapping{}
	for _, slice := range slices {
		var service map[string][]podEndPoint
		var hasService bool
		if service, hasService = mapping[slice.Service]; !hasService {
			service = map[string][]podEndPoint{}
			mapping[slice.Service] = service
		}
		for _, e := range slice.Endpoints {
			for _, address := range e.Addresses {
				for _, port := range slice.Ports {
					service[e.Topology.Zone] = append(service[e.Topology.Zone], podEndPoint{
						IP:   address,
						Port: port.Port,
						Zone: e.Topology.Zone,
					})
				}
			}
		}
	}
	return mapping
}

// Watch always emits the last computed value first, so the consumer can start immediately
func (d *DiscoveryImpl) Watch() <-chan Mapping {
	d.Lock()
	defer d.Unlock()

	var size = 0
	if d.last != nil {
		size = 1
	}
	ch := make(chan Mapping, size)
	if d.last != nil {
		ch <- d.last
	}
	d.workers = append(d.workers, func(m Mapping) {
		ch <- m
	})
	return ch
}

func (d *DiscoveryImpl) Emit(m Mapping) {
	d.Lock()
	defer d.Unlock()
	d.last = m
	for _, w := range d.workers {
		w(m)
	}
}

// MockDiscovery 'discovers' from a file like mapping.yaml
type MockDiscovery struct {
	filePath string
	DiscoveryImpl
}

func (d *MockDiscovery) Start(ctx context.Context, upstreamServices []string) error {
	initialStat, err := os.Stat(d.filePath)
	if err != nil {
		return err
	}

	read := func() (dst Mapping) {
		data, _ := ioutil.ReadFile(d.filePath)
		err = yaml.Unmarshal(data, &dst)
		if err != nil {
			zap.S().Warnf("Invalid format for Mapping in file %s", d.filePath)
		}
		return
	}

	initial := read()
	d.Emit(initial)

	t := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			stat, err := os.Stat(d.filePath)
			if err != nil {
				d.Emit(nil)
				return err
			}
			if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
				d.Emit(read())
			}
		}
	}
}

package internal

import (
	"context"
	"sync"
	"time"

	"github.com/bep/debounce"
	"k8s.io/apimachinery/pkg/watch"
)

type Discovery struct {
	sync.Mutex
	last    Mapping
	workers []func(Mapping)
}

func (d *Discovery) Start(ctx context.Context, upstreamServices []string) {
	slices := make(map[string]Slice)
	debounced := debounce.New(50 * time.Millisecond)
	dowatch(ctx, func(t watch.EventType, s Slice) {
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

func (d *Discovery) computeMapping(slices map[string]Slice) Mapping {
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

func (d *Discovery) Watch() <-chan Mapping {
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

func (d *Discovery) Emit(m Mapping) {
	d.Lock()
	defer d.Unlock()
	d.last = m
	for _, w := range d.workers {
		w(m)
	}
}

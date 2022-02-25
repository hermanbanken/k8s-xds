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

type Discovery struct {
	sync.Mutex
	last    Mapping
	workers []func(Mapping)
	fn      func(context.Context, func(t watch.EventType, s Slice)) error
}

func (d *Discovery) Start(ctx context.Context, upstreamServices []string) error {
	slices := make(map[string]Slice)
	debounced := debounce.New(50 * time.Millisecond)
	return d.fn(ctx, func(t watch.EventType, s Slice) {
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

func FileWatch(ctx context.Context, fn func(t watch.EventType, s Slice)) error {
	filePath := "slice.yaml"
	if env, hasEnv := os.LookupEnv("TEST_SLICE_FILE"); hasEnv {
		filePath = env
	}
	initialStat, err := os.Stat(filePath)
	if err != nil {
		return err
	}

	read := func() (dst Slice) {
		data, _ := ioutil.ReadFile(filePath)
		err = yaml.Unmarshal(data, &dst)
		if err != nil {
			zap.S().Warnf("Invalid format for Slice in file %s", filePath)
		}
		return Slice{}
	}

	initial := read()
	fn(watch.Added, initial)

	t := time.NewTicker(500 * time.Millisecond)
	for {
		select {
		case <-ctx.Done():
			return nil
		case <-t.C:
			stat, err := os.Stat(filePath)
			if err != nil {
				fn(watch.Deleted, Slice{Name: initial.Name})
				return err
			}
			if stat.Size() != initialStat.Size() || stat.ModTime() != initialStat.ModTime() {
				fn(watch.Modified, read())
			}
		}
	}
}

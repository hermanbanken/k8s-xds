package internal

import (
	v1 "k8s.io/api/discovery/v1"
	"k8s.io/api/discovery/v1beta1"
)

func (slice *Slice) FromV1Beta1(es *v1beta1.EndpointSlice) {
	slice.Name = es.GetName()
	slice.Service = es.GetLabels()["kubernetes.io/service-name"]
	slice.AddressType = string(es.AddressType)
	slice.Endpoints = make([]Endpoint, len(es.Endpoints))
	slice.Ports = make([]Port, len(es.Ports))
	for i, e := range es.Endpoints {
		slice.Endpoints[i].FromK8s(e.Addresses, e.Conditions.Ready, e.Hostname, nil, nil)
		slice.Endpoints[i].Topology.Host = e.Topology["kubernetes.io/hostname"]
		slice.Endpoints[i].Topology.Zone = e.Topology["topology.kubernetes.io/zone"]
	}
	for i, p := range es.Ports {
		slice.Ports[i].FromK8s(p.Name, p.Port, (*string)(p.Protocol))
	}
}

func (slice *Slice) FromV1(es *v1.EndpointSlice) {
	slice.Name = es.GetName()
	slice.Service = es.GetLabels()["kubernetes.io/service-name"]
	slice.AddressType = string(es.AddressType)
	slice.Endpoints = make([]Endpoint, len(es.Endpoints))
	slice.Ports = make([]Port, len(es.Ports))
	for i, e := range es.Endpoints {
		slice.Endpoints[i].FromK8s(e.Addresses, e.Conditions.Ready, e.Hostname, e.NodeName, e.Zone)
	}
	for i, p := range es.Ports {
		slice.Ports[i].FromK8s(p.Name, p.Port, (*string)(p.Protocol))
	}
}

type Slice struct {
	Name        string
	Service     string
	AddressType string // IPv4 IPv6
	Endpoints   []Endpoint
	Ports       []Port
}
type Endpoint struct {
	Addresses  []string
	Ready      bool
	TargetName string
	Topology   Topology
}

func (e *Endpoint) FromK8s(addr []string, ready *bool, targetName *string, host *string, zone *string) {
	e.Addresses = addr
	if ready != nil {
		e.Ready = *ready
	}
	if targetName != nil {
		e.TargetName = *targetName
	}
	if host != nil {
		e.Topology.Host = *host
	}
	if zone != nil {
		e.Topology.Zone = *zone
	}
}

type Topology struct {
	Host string
	Zone string
}
type Port struct {
	Name     string
	Protocol string
	Port     int32
}

func (port *Port) FromK8s(Name *string, Port *int32, Protocol *string) {

	if Name != nil {
		port.Name = *Name
	}
	if Port != nil {
		port.Port = *Port
	}
	if Protocol != nil {
		port.Protocol = string(*Protocol)
	}
}

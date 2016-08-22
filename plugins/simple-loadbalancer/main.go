package main

import (
	"log"
	"math/rand"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/gatekeeper/utils"
	loadbalancer_plugin "github.com/jonmorehouse/gatekeeper/plugin/loadbalancer"
)

// implements the loadbalancer.LoadBalancer plugin that is exposed over RPC
type LoadBalancer struct {
	services utils.ServiceContainer
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		services: utils.NewServiceContainer(),
	}
}

func (l *LoadBalancer) Start() error                                           { return nil }
func (l *LoadBalancer) Stop() error                                            { return nil }
func (l *LoadBalancer) Configure(opts map[string]interface{}) error            { return nil }
func (l *LoadBalancer) UpstreamMetric(metric *gatekeeper.UpstreamMetric) error { return nil }
func (l *LoadBalancer) Heartbeat() error {
	log.Println("simple loadbalancer heartbeat ...")
	return nil
}

func (l *LoadBalancer) AddBackend(upstreamID gatekeeper.UpstreamID, backend *gatekeeper.Backend) error {
	return l.services.AddBackend(upstreamID, backend)
}

func (l *LoadBalancer) RemoveBackend(backend *gatekeeper.Backend) error {
	return l.services.RemoveBackend(backend.ID)
}

func (l *LoadBalancer) GetBackend(upstreamID gatekeeper.UpstreamID) (*gatekeeper.Backend, error) {
	backends, err := l.services.FetchBackends(upstreamID)
	if err != nil {
		return nil, err
	}
	return backends[rand.Intn(len(backends))], nil
}

func main() {
	rand.Seed(time.Now().Unix())
	loadBalancer := NewLoadBalancer()
	if err := loadbalancer_plugin.RunPlugin("simple-loadbalancer", loadBalancer); err != nil {
		log.Fatal(err)
	}
}

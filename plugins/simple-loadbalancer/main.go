package main

import (
	"fmt"
	"log"
	"math/rand"
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	loadbalancer_plugin "github.com/jonmorehouse/gatekeeper/plugin/loadbalancer"
)

// implements the loadbalancer.LoadBalancer plugin that is exposed over RPC
type LoadBalancer struct {
	sync.RWMutex

	upstreamBackends map[gatekeeper.UpstreamID][]*gatekeeper.Backend
}

func NewLoadBalancer() *LoadBalancer {
	return &LoadBalancer{
		upstreamBackends: make(map[gatekeeper.UpstreamID][]*gatekeeper.Backend),
	}
}

// no special configuration needed, but we implement these methods anyway for the interface
func (l *LoadBalancer) Start() error {
	log.Println("simple-loadbalancer plugin started...")
	return nil
}

func (l *LoadBalancer) Stop() error {
	log.Println("simple-loadbalancer plugin stopped...")
	return nil
}

func (l *LoadBalancer) Configure(opts map[string]interface{}) error {
	log.Println("configuring simple-loadbalancer ...")
	return nil
}

func (l *LoadBalancer) Heartbeat() error {
	log.Println("simple loadbalancer heartbeat ...")
	return nil
}

// actual implementation of methods used
func (l *LoadBalancer) AddBackend(upstream gatekeeper.UpstreamID, backend *gatekeeper.Backend) error {
	l.Lock()
	defer l.Unlock()
	log.Println("add backend")
	log.Println(upstream, backend)

	if _, ok := l.upstreamBackends[upstream]; !ok {
		l.upstreamBackends[upstream] = make([]*gatekeeper.Backend, 0, 1)
	}
	l.upstreamBackends[upstream] = append(l.upstreamBackends[upstream], backend)
	return nil
}

func (l *LoadBalancer) RemoveBackend(deleted *gatekeeper.Backend) error {
	l.Lock()
	defer l.Unlock()
	found := false

	for upstream, backends := range l.upstreamBackends {
		for idx, backend := range backends {
			if backend != deleted {
				continue
			}

			found = true
			backends = append(backends[:idx], backends[idx+1:]...)
			l.upstreamBackends[upstream] = backends
			break
		}
	}

	if !found {
		return fmt.Errorf("Did not find backend")
	}
	return nil
}

func (l *LoadBalancer) UpstreamMetric(metric *gatekeeper.UpstreamMetric) error {
	log.Println("upstream metric ...")
	return nil
}

func (l *LoadBalancer) GetBackend(upstream gatekeeper.UpstreamID) (*gatekeeper.Backend, error) {
	l.RLock()
	defer l.RUnlock()
	backends, found := l.upstreamBackends[upstream]
	if !found {
		return nil, fmt.Errorf("UPSTREAM_NOT_FOUND")
	} else if len(backends) == 0 {
		return nil, fmt.Errorf("NO_BACKENDS_FOUND")
	}

	// pick a random backend for this upstream and return it
	idx := rand.Intn(len(backends))
	return backends[idx], nil
}

func main() {
	rand.Seed(time.Now().Unix())
	loadBalancer := NewLoadBalancer()
	if err := loadbalancer_plugin.RunPlugin("simple-loadbalancer", loadBalancer); err != nil {
		log.Fatal(err)
	}
}

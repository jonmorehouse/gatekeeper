package main

import (
	"log"
	"sync"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	router_plugin "github.com/jonmorehouse/gatekeeper/plugin/router"
)

// router implements the `router_plugin.Plugin` interface
func NewRouter() router_plugin.Plugin {
	return &router{
		upstreams: make(map[gatekeeper.UpstreamID]*gatekeeper.Upstream),
	}
}

type router struct {
	upstreams map[gatekeeper.UpstreamID]*gatekeeper.Upstream

	sync.RWMutex
}

func (r *router) Start() error { return nil }
func (r *router) Stop() error  { return nil }
func (r *router) Heartbeat() error {
	log.Println("router heartbeat received")
	return nil
}
func (r *router) Configure(opts map[string]interface{}) error { return nil }

func (r *router) AddUpstream(upstream *gatekeeper.Upstream) error {
	log.Println("adding upstream ....")
	r.Lock()
	defer r.Unlock()

	if _, ok := r.upstreams[upstream.ID]; !ok {
		log.Println("adding new upstream to route table")
	} else {
		log.Println("updating upstream in route table")
	}

	r.upstreams[upstream.ID] = upstream
	return nil
}

func (r *router) RemoveUpstream(upstreamID gatekeeper.UpstreamID) error {
	r.Lock()
	defer r.Unlock()

	if _, ok := r.upstreams[upstreamID]; ok {
		log.Println("removing upstream from route table")
	}

	delete(r.upstreams, upstreamID)
	return nil
}

func (r *router) RouteRequest(req *gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, error) {
	log.Println("route request called ")
	r.RLock()
	defer r.RUnlock()

	for _, upstream := range r.upstreams {
		for _, hostname := range upstream.Hostnames {
			if req.Host == hostname {
				req.UpstreamMatchType = gatekeeper.HostnameUpstreamMatch
				return upstream, req, nil
			}
		}

		for _, prefix := range upstream.Hostnames {
			if req.Prefix == prefix {
				req.Path = req.PrefixlessPath
				req.UpstreamMatchType = gatekeeper.PrefixUpstreamMatch
				return upstream, req, nil
			}
		}
	}

	return nil, nil, nil
}

func main() {
	router := NewRouter()
	if err := router_plugin.RunPlugin("example-router", router); err != nil {
		log.Fatal(err)
	}
}

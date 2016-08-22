package main

import (
	"log"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/gatekeeper/utils"
	router_plugin "github.com/jonmorehouse/gatekeeper/plugin/router"
)

// router implements the `router_plugin.Plugin` interface
func NewRouter() router_plugin.Plugin {
	return &router{utils.NewUpstreamContainer()}
}

type router struct {
	upstreams utils.UpstreamContainer
}

func (r *router) Configure(opts map[string]interface{}) error { return nil }
func (r *router) Start() error                                { return nil }
func (r *router) Stop() error                                 { return nil }
func (r *router) Heartbeat() error {
	log.Println("router heartbeat received")
	return nil
}

// add an upstream into the upstream container
func (r *router) AddUpstream(upstream *gatekeeper.Upstream) error {
	return r.upstreams.AddUpstream(upstream)
}

// remove an upstream from the upstream container
func (r *router) RemoveUpstream(upstreamID *gatekeepr.UpstreamID) error {
	return r.upstreams.RemoveUpstream(upstreamID)
}

// route a request, returning the correct upstream if one matches for the request
func (r *router) RouteRequest(req *gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, error) {
	upstream, err := r.upstreams.UpstreamsByPrefix(req.Prefix)
	if err == nil {
		req.Path = req.PrefixlessPath
		req.UpstreamMatchType = PrefixUpstreamMatch
		return upstream, req, nil
	}

	upstream, err = r.upstreams.UpstreamsByHostname(req.Host)
	if err == nil {
		req.UpstreamMatchType = HostnameUpstreamMatch
		return upstream, req, nil
	}

	return nil, req, gatekeeper.UpstreamNotFoundErr
}

func main() {
	router := NewRouter()
	if err := router_plugin.RunPlugin("router", router); err != nil {
		log.Fatal(err)
	}
}

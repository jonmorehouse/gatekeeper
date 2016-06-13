package main

import (
	"log"
	"time"

	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type StaticUpstreams struct {
	manager upstream_plugin.Manager
	stopCh  chan interface{}
}

func (s *StaticUpstreams) Configure(map[string]interface{}) error {
	return nil
}

func (s *StaticUpstreams) Heartbeat() error {
	return nil
}

func (s *StaticUpstreams) Start(manager upstream_plugin.Manager) error {
	log.Println("static-upstreams plugin started...")
	s.manager = manager
	go s.worker()
	return nil
}

func (s *StaticUpstreams) Stop() error {
	log.Println("static-upstreams plugin stopped...")
	return nil
}

func (s *StaticUpstreams) worker() {
	upstream := &shared.Upstream{
		ID:        shared.NewUpstreamID(),
		Name:      "httpbin",
		Protocols: []shared.Protocol{shared.HTTPPublic, shared.HTTPInternal},
		Prefixes:  []string{"httpbin"},
		Hostnames: []string{"httpbin.org", "httpbin"},
		Timeout:   time.Second * 5,
	}
	err := s.manager.AddUpstream(upstream)
	if err != nil {
		log.Fatal(err)
	}

	backend := &shared.Backend{
		ID:      shared.NewBackendID(),
		Address: "https://httpbin.org",
	}

	err = s.manager.AddBackend(upstream.ID, backend)
	if err != nil {
		log.Println("Static upstreams plugin was unable to emit a backend")
		log.Println(err)
	}

	// block in this background worker until a stop signal is triggered by
	// the parent; periodically, we re-add all of the upstreams we know of.
	for {
		select {
		case <-s.stopCh:
			return
		default:
			time.Sleep(time.Second)
		}
	}
}

func main() {
	staticUpstreams := StaticUpstreams{
		stopCh: make(chan interface{}),
	}

	if err := upstream_plugin.RunPlugin("static-upstreams", &staticUpstreams); err != nil {
		log.Fatal(err)
	}
}

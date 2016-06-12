package main

import (
	"log"
	"time"

	plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type StaticUpstreams struct {
	manager plugin.Manager
	stopCh  chan interface{}
}

func (s *StaticUpstreams) Configure(map[string]interface{}) *shared.Error {
	return nil
}

func (s *StaticUpstreams) Heartbeat() *shared.Error {
	return nil
}

func (s *StaticUpstreams) Start(manager plugin.Manager) *shared.Error {
	log.Println("static-upstreams plugin started...")
	s.manager = manager
	go s.worker()
	return nil
}

func (s *StaticUpstreams) Stop() *shared.Error {
	log.Println("static-upstreams plugin stopped...")
	return nil
}

func (s *StaticUpstreams) worker() {
	upstr := shared.Upstream{
		ID:        shared.NewUpstreamID(),
		Name:      "httpbin",
		Prefixes:  []string{"httpbin"},
		Hostnames: []string{"httpbin.org", "httpbin"},
	}
	err := s.manager.AddUpstream(upstr)
	if err != nil {
		log.Fatal(err)
	}

	backend := shared.Backend{
		ID:      shared.NewBackendID(),
		Address: "https://httpbin.org",
	}

	err = s.manager.AddBackend(upstr.ID, backend)
	if err != nil {
		log.Println("Static upstreams plugin was unable to emit a backend")
		log.Println(err)
	}

	// hang around until the process exits!
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
	if err := plugin.RunPlugin("static-upstreams", &staticUpstreams); err != nil {
		log.Fatal(err)
	}
}

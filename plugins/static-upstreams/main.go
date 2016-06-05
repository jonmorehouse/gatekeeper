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

func (s *StaticUpstreams) Configure(map[string]interface{}) error {
	return nil
}

func (s *StaticUpstreams) Heartbeat() error {
	return nil
}

func (s *StaticUpstreams) Start(manager plugin.Manager) error {
	s.manager = manager
	go s.worker()
	return nil
}

func (s *StaticUpstreams) Stop() error {
	return nil
}

func (s *StaticUpstreams) worker() {
	upstr := shared.Upstream{
		ID:   shared.NewUpstreamID(),
		Name: "httpbin",
	}
	err := s.manager.AddUpstream(upstr)
	if err != nil {
		log.Fatal(err)
	}

	backend := shared.Backend{
		Address: "https://httpbin.org",
	}

	err = s.manager.AddBackend(upstr.ID, backend)
	if err != nil {
		log.Fatal(err)
	}

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

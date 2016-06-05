package main

import (
	"log"
	"time"

	"github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type StaticUpstreams struct {
	manager upstream.Manager
	stopCh  chan interface{}
}

func (s *StaticUpstreams) Configure(opts upstream.Opts) error {
	return nil
}

func (s *StaticUpstreams) Heartbeat() error {
	return nil
}

func (s *StaticUpstreams) Start(manager upstream.Manager) error {
	s.manager = manager
	go s.worker()
	return nil
}

func (s *StaticUpstreams) Stop() error {
	return nil
}

func (s *StaticUpstreams) worker() {
	upstr := upstream.Upstream{
		ID:   upstream.NilUpstreamID,
		Name: "httpbin",
	}
	upstrID, err := s.manager.AddUpstream(upstr)
	if err != nil {
		log.Fatal(err)
	}
	upstr.ID = upstrID

	backend := upstream.Backend{
		Address: "https://httpbin.org",
	}
	_, err = s.manager.AddBackend(upstrID, backend)
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
	if err := upstream.RunPlugin("static-upstreams", &staticUpstreams); err != nil {
		log.Fatal(err)
	}
}

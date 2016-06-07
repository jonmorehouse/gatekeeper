package gatekeeper

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
	"github.com/tylerb/graceful"
)

type Server interface {
	Start() error
	StopAccepting() error
	Stop(time.Duration) error
}

type ProxyServer struct {
	port              uint
	protocol          shared.Protocol
	upstreamRequester UpstreamRequesterClient
	loadBalancer      LoadBalancerClient
	requestModifier   RequestModifierClient
	responseModifier  ResponseModifierClient

	stopAccepting bool
	stopFn        func(time.Duration) error

	httpServer *graceful.Server
}

func (s *ProxyServer) Start() error {
	var starter func() error
	var err error

	if s.protocol == shared.HTTPPublic || s.protocol == shared.HTTPPrivate {
		starter, err = s.startHTTP()
	} else if s.protocol == shared.TCPPublic || s.protocol == shared.TCPPrivate {
		starter, err = s.startTCP()
	} else {
		return fmt.Errorf("Invalid protocol")
	}

	if err != nil {
		return err
	}

	return starter()
}

func (s *ProxyServer) StopAccepting() error {
	s.stopAccepting = true
	return nil
}

func (s *ProxyServer) Stop(duration time.Duration) error {
	if err := s.stopFn(time.Second); err != nil {
		return err
	}
	return nil
}

func (s *ProxyServer) startHTTP() (func() error, error) {
	mux := http.NewServeMux()
	mux.HandleFunc("/", s.httpHandler)

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", s.port),
		Handler: mux,
	}

	s.httpServer = &graceful.Server{
		Server:           server,
		NoSignalHandling: true,
	}

	s.stopFn = func(dur time.Duration) error {
		log.Println("shutting down http server")
		s.httpServer.Stop(dur)
		return nil
	}

	return func() error {
		log.Println("starting http server and listening on port ", s.port)
		return s.httpServer.ListenAndServe()
	}, nil
}

func (s *ProxyServer) httpHandler(rw http.ResponseWriter, req *http.Request) {
	if s.stopAccepting {
		io.WriteString(rw, "server is shutting down")
	}

	io.WriteString(rw, "hello world")
}

func (s *ProxyServer) startTCP() (func() error, error) {
	s.stopFn = func(time.Duration) error { return nil }
	return func() error {
		return nil
	}, nil
}

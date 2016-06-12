package gatekeeper

import (
	"fmt"
	"io"
	"net/http"
	"net/http/httputil"
	"net/url"
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

	httpServer *graceful.Server
	errCh      chan error
}

func (s *ProxyServer) Start() error {
	return s.startHTTP()
}

func (s *ProxyServer) StopAccepting() error {
	s.stopAccepting = true
	return nil
}

func (s *ProxyServer) Stop(duration time.Duration) error {
	s.httpServer.Stop(duration)
	return <-s.errCh
}

func (s *ProxyServer) startHTTP() error {
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

	// the errCh is responsible for emitting an error when the server fails or closes.
	errCh := make(chan error, 1)

	// start the server in a goroutine, passing any errors back to the errCh
	go func() {
		err := s.httpServer.ListenAndServe()
		errCh <- err
	}()

	// now we wait a maximum of 100milliseconds, which is arbitrary to
	// catch any immediate errors from the listener
	timeout := time.Now().Add(time.Millisecond * 100)
	for {
		select {
		case err := <-errCh:
			return err
		default:
			if time.Now().After(timeout) {
				goto finished
			}
		}
	}

finished:
	s.errCh = errCh
	return nil
}

func (s *ProxyServer) httpHandler(rw http.ResponseWriter, req *http.Request) {
	if s.stopAccepting {
		io.WriteString(rw, "SERVER_SHUTTING_DOWN")
		return
	}

	upstream, err := s.upstreamRequester.UpstreamForRequest(req)
	if err != nil {
		io.WriteString(rw, "NO_UPSTREAM_FOUND")
		return
	}

	backend, err := s.loadBalancer.GetBackend(upstream)
	if err != nil {
		io.WriteString(rw, "NO_BACKEND_FOUND")
		return
	}

	backendURL, err := url.Parse(backend.Address)
	if err != nil {
		io.WriteString(rw, "INVALID_BACKEND_URL")
		return
	}

	PrepareBackendRequest(req, upstream, backend)
	proxy := httputil.NewSingleHostReverseProxy(backendURL)
	proxy.ServeHTTP(rw, req)
}

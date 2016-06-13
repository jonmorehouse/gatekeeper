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
	modifier          Modifier

	stopAccepting bool

	httpServer *graceful.Server
	proxier    Proxier
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
		log.Println("listening on port: ", s.port)
		err := s.httpServer.ListenAndServe()
		errCh <- err
	}()

	for {
		select {
		case err := <-errCh:
			return err
		case <-time.After(time.Millisecond * 100):
			goto finished
		}
	}

finished:
	s.errCh = errCh
	return nil
}

func (s *ProxyServer) httpHandler(rw http.ResponseWriter, rawReq *http.Request) {
	if s.stopAccepting {
		io.WriteString(rw, "SERVER_SHUTTING_DOWN")
		return
	}

	// build a *shared.Request for this rawReq; a wrapper with additional
	// meta information around an *http.Request object
	req := shared.NewRequest(rawReq, s.protocol)
	upstream, matchType, err := s.upstreamRequester.UpstreamForRequest(req)
	if err != nil {
		io.WriteString(rw, "NO_UPSTREAM_FOUND")
		return
	}

	req.Upstream = upstream
	req.UpstreamMatchType = matchType

	backend, err := s.loadBalancer.GetBackend(upstream)
	if err != nil {
		io.WriteString(rw, "NO_BACKEND_FOUND")
		return
	}

	req, err = s.modifier.ModifyRequest(req)
	if err != nil {
		log.Println(err)
		io.WriteString(rw, "FAILURE_TO_MODIFY_REQUEST")
		return
	}
	if req.Err != nil {
		io.WriteString(rw, "request modifier failed")
		return
	}

	// pass the request along to the proxier to perform the request
	// lifecycle on to the backend.
	if err := s.proxier.Proxy(rw, rawReq, req, backend); err != nil {
		io.WriteString(rw, "UNABLE_TO_PROXY")
		return
	}
}

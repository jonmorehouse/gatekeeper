package gatekeeper

import (
	"io"
	"net/http"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type Server interface {
	Start() error
	ProxyHTTP(http.ResponseWriter, *http.Request) error
	Stop(time.Duration) error
}

type ProxyServer struct {
	protocol         shared.Protocol
	upstreams        UpstreamRequester
	loadBalancer     LoadBalancer
	requestModifier  RequestModifier
	responseModifier ResponseModifier
}

func NewProxyServer() Server {
	return &ProxyServer{
		protocol: shared.HTTPPublic,
	}
}

func (s *ProxyServer) Start() error {
	return nil
}

func (s *ProxyServer) Stop(time.Duration) error {
	return nil
}

func (s *ProxyServer) ProxyHTTP(rw http.ResponseWriter, req *http.Request) error {
	upstream, _ := s.upstreams.UpstreamForRequest(req)
	backend, _ := s.loadBalancer.GetBackend(upstream)

	io.WriteString(rw, backend.Address)
	return nil
}

func (s *ProxyServer) ProxyTCP() error {
	panic("not yet implemented...")
	return nil
}

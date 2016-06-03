package gatekeeper

import (
	"net/http"
	"time"
)

type Server interface {
	Start() error
	HandleHTTP(http.ResponseWriter, *http.Request) error
	Stop(time.Duration) error
}

type ServerOpts struct {
	ListenProtocol
}

type ProxyServer struct {
	Protocol          ProtocolType
	UpstreamRequester UpstreamRequester
	BackendRequester  BackendRequester

	// TODO add in the concept of request modifiers so that users can
	// modify requests in the manner that they like
	// RequestModifiers  []RequestModifiers
	// ResponseModifiers []ResponseModifiers
}

func NewProxyServer() Server {
	return &ProxyServer{
		Protocols: []ProtocolType{HTTPPublic, HTTPPrivate},
	}
}

func (p *ProxyServer) Start() error {
	return nil
}

func (p *ProxyServer) Stop(time.Duration) error {
	return nil
}

func (p *ProxyServer) ProxyHTTP(rw http.ResponseWeriter, req *http.Request) error {
	// pass
}

// TODO figure out the ProxyTCP signature
func (p *ProxyServer) ProxyTCP() error {

}

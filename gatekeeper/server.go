package gatekeeper

import (
	"fmt"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type Server interface {
	Start() error
	Stop(time.Duration) error
}

type ProxyServer struct {
	port              uint
	protocol          shared.Protocol
	upstreamRequester UpstreamRequesterClient
	loadBalancer      LoadBalancerClient
	requestModifier   RequestModifierClient
	responseModifier  ResponseModifierClient
}

func (s *ProxyServer) Start() error {
	fmt.Println("proxy server started ...")
	return nil
}

func (s *ProxyServer) Stop(time.Duration) error {
	fmt.Println("proxy server stopped...")
	return nil
}

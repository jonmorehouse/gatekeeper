package gatekeeper

import (
	"fmt"
	"log"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
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
	stopCh            chan interface{}
}

func (s *ProxyServer) Start() error {
	log.Println("server started...")
	for {
		select {
		case <-s.stopCh:
			return nil
		default:
			time.Sleep(time.Second)
		}
	}
	return nil
}

func (s *ProxyServer) StopAccepting() error {
	return nil
}

func (s *ProxyServer) Stop(time.Duration) error {
	fmt.Println("proxy server stopped...")
	s.stopCh <- struct{}{}
	return nil
}

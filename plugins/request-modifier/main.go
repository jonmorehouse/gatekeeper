package main

import (
	"fmt"
	"log"

	request_plugin "github.com/jonmorehouse/gatekeeper/plugin/request"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type Plugin struct {
	counter uint
}

func (*Plugin) Configure(args map[string]interface{}) error {
	log.Println("configuring request modifier plugin")
	return nil
}

func (*Plugin) Heartbeat() error {
	log.Println("request-modifier plugin heartbeat")
	return nil
}

func (*Plugin) Start() error {
	log.Println("calling Start() on request-modifier plugin")
	return nil
}

func (*Plugin) Stop() error {
	log.Println("calling Stop() on request-modifier plugin")
	return nil
}

func (p *Plugin) ModifyRequest(req *shared.Request) (*shared.Request, error) {
	if req.Header == nil {
		req.Header = make(map[string][]string)
	}
	p.counter += 1
	req.Header["X-Request-ID"] = []string{fmt.Sprintf("%d", p.counter)}
	log.Println("adding X-Request-ID header to the request")
	return req, nil
}

func main() {
	if err := request_plugin.RunPlugin("request-modifier", &Plugin{}); err != nil {
		log.Fatal(err)
	}
}

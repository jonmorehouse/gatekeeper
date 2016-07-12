package main

import (
	"fmt"
	"log"

	modifier_plugin "github.com/jonmorehouse/gatekeeper/plugin/modifier"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type Plugin struct {
	counter uint
}

func (*Plugin) Configure(args map[string]interface{}) error {
	log.Println("configuring modifier plugin")
	return nil
}

func (*Plugin) Heartbeat() error {
	log.Println("modifier plugin heartbeat")
	return nil
}

func (*Plugin) Start() error {
	log.Println("calling Start() on modifier plugin")
	return nil
}

func (*Plugin) Stop() error {
	log.Println("calling Stop() on modifier plugin")
	return nil
}

func (p *Plugin) ModifyRequest(req *gatekeeper.Request) (*gatekeeper.Request, error) {
	if req.Header == nil {
		req.Header = make(map[string][]string)
	}
	p.counter += 1
	req.Header["X-Request-ID"] = []string{fmt.Sprintf("%d", p.counter)}
	return req, nil
}

func (Plugin) ModifyResponse(req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, error) {
	resp.Header.Set("X-Request-ID", req.Header["X-Request-ID"][0])
	return resp, nil
}

func (Plugin) ModifyErrorResponse(err error, req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, error) {
	//resp.Header.Add("X-Error", "error")
	return resp, nil
}

func main() {
	if err := modifier_plugin.RunPlugin("modifier", &Plugin{}); err != nil {
		log.Fatal(err)
	}
}

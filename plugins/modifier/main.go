package main

import (
	"log"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	modifier_plugin "github.com/jonmorehouse/gatekeeper/plugin/modifier"
)

func newPlugin() modifier_plugin.Plugin {
	return plugin{}
}

type plugin struct{}

func (p plugin) Start() error                           { return nil }
func (p plugin) Stop() error                            { return nil }
func (p plugin) Heartbeat() error                       { return nil }
func (p plugin) Configure(map[string]interface{}) error { return nil }
func (p plugin) ModifyErrorResponse(err error, req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, error) {
	return resp, nil
}

func (p plugin) ModifyRequest(req *gatekeeper.Request) (*gatekeeper.Request, error) {
	if req.Header.Get("X-Request-ID") == "" {
		req.Header.Set("X-Request-ID", gatekeeper.GetUUID())
	}
	return req, nil
}

func (p plugin) ModifyResponse(req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, error) {
	if resp.Header.Get("X-Request-ID") == "" {
		resp.Header.Set("X-Request-ID", req.Header.Get("X-Request-ID"))
	}
	return nil, nil
}

func main() {
	plugin := newPlugin()
	if err := modifier_plugin.RunPlugin("modifier", plugin); err != nil {
		log.Fatal(err)
	}
}

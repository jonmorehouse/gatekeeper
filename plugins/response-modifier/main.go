package main

import (
	"log"
	"strings"

	response_plugin "github.com/jonmorehouse/gatekeeper/plugin/response"
	"github.com/jonmorehouse/gatekeeper/shared"
)

// Plugin is a type that implements the response_plugin.Plugin interface
type Plugin struct{}

func (Plugin) Start() error                           { return nil }
func (Plugin) Stop() error                            { return nil }
func (Plugin) Heartbeat() error                       { return nil }
func (Plugin) Configure(map[string]interface{}) error { return nil }

func (Plugin) ModifyResponse(req *shared.Request, resp *shared.Response) (*shared.Response, error) {
	resp.Header.Set("X-Request-ID", req.Header["X-Request-ID"][0])

	// replace the body with a custom message
	reader := strings.NewReader("hello world this was overwritten in a plugin")
	if err := resp.SetBody(reader); err != nil {
		return resp, err
	}

	return resp, nil
}

func main() {
	if err := response_plugin.RunPlugin("response-modifier", &Plugin{}); err != nil {
		log.Fatal(err)
	}
}

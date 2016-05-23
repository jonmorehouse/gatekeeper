package gatekeeper

import (
	"testing"
	"time"
)

func Test__temp(t *testing.T) {
	pluginOpts := PluginOpts{
		Name: "static-upstreams",
		Cmd:  "plugin-static-upstreams",
		Opts: map[string]interface{}{
			"test": "test",
		},
	}
	um, err := NewUpstreamManager(pluginOpts)
	if err != nil {
		t.Fatalf(err.Error())
	}

	err = um.Start()
	if err != nil {
		t.Fatalf(err.Error())
	}

	t.Log("progress...")
	time.Sleep(time.Second * 1)
	upstreamManager := um.(*UpstreamManager)
	upstr, backend := upstreamManager.TempFetch()
	if upstr.Name != "httpbin" {
		t.Fatalf("Invalid name")
	}
	if backend.Address != "http://httpbin.org" {
		t.Fatalf("Invalid address")
	}
	if err := um.Stop(); err != nil {
		t.Fatalf("Stop erred out...")
	}
}

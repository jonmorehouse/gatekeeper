package gatekeeper

import (
	"fmt"
	"sync"
	"time"

	response_plugin "github.com/jonmorehouse/gatekeeper/plugin/response"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type ResponseModifier interface {
	Start() error
	Stop(time.Duration) error
	ModifyResponse(*shared.Request, *shared.Response) (*shared.Response, error)
}

type ResponseModifierClient interface {
	ModifyResponse(*shared.Request, *shared.Response) (*shared.Response, error)
}

type responseModifier struct {
	pluginManagers []PluginManager
}

func NewResponseModifier(pluginManagers []PluginManager) ResponseModifier {
	return &responseModifier{
		pluginManagers: pluginManagers,
	}
}

func (r *responseModifier) Start() error {
	errs := NewAsyncMultiError()
	var wg sync.WaitGroup

	// start all instances of all plugins
	for _, manager := range r.pluginManagers {
		wg.Add(1)

		go func(manager PluginManager) {
			defer wg.Done()
			if err := manager.Start(); err != nil {
				errs.Add(err)
			}
		}(manager)
	}

	wg.Wait()
	return errs.ToErr()
}

func (r *responseModifier) Stop(duration time.Duration) error {
	errs := NewAsyncMultiError()

	var wg sync.WaitGroup
	doneCh := make(chan interface{})

	for _, manager := range r.pluginManagers {
		wg.Add(1)
		go func(p PluginManager) {
			defer wg.Done()
			if err := p.Stop(duration); err != nil {
				errs.Add(err)
			}
		}(manager)
	}

	// wait for all of the plugin instances to stop
	go func() {
		wg.Wait()
		doneCh <- struct{}{}
	}()

	for {
		select {
		case <-doneCh:
			return errs.ToErr()
		case <-time.After(duration):
			errs.Add(fmt.Errorf("timeout waiting for response plugins to Stop"))
			return errs.ToErr()
		}
	}

	return nil
}

func (r *responseModifier) ModifyResponse(req *shared.Request, resp *shared.Response) (*shared.Response, error) {
	var err error

	for _, pluginManager := range r.pluginManagers {
		plugin, err := pluginManager.Get()
		if err != nil {
			return resp, fmt.Errorf("Unable to get response modifier plugin")
		}

		responsePlugin, ok := plugin.(response_plugin.PluginRPC)
		if !ok {
			return resp, fmt.Errorf("INVALID_RESPONSE_PLUGIN_TYPE")
		}

		resp, err = responsePlugin.ModifyResponse(req, resp)
		if err != nil {
			break
		}
	}

	return resp, err
}

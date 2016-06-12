package gatekeeper

import (
	"fmt"
	"sync"
	"time"

	request_plugin "github.com/jonmorehouse/gatekeeper/plugin/request"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type RequestModifier interface {
	Start() error
	Stop(time.Duration) error
	ModifyRequest(*shared.Request) (*shared.Request, error)
}

type RequestModifierClient interface {
	ModifyRequest(*shared.Request) (*shared.Request, error)
}

type requestModifier struct {
	pluginManagers []PluginManager
}

func NewRequestModifier(pluginManagers []PluginManager) RequestModifier {
	return &requestModifier{
		pluginManagers: pluginManagers,
	}
}

func (r *requestModifier) Start() error {
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

func (r *requestModifier) Stop(duration time.Duration) error {
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

	// wait for all managers to stop
	go func() {
		wg.Wait()
		doneCh <- struct{}{}
	}()

	// wait for the waitGroup to finish or handle the timeout with an error
	for {
		select {
		case <-doneCh:
			return errs.ToErr()
		case <-time.After(duration):
			errs.Add(fmt.Errorf("timeout waiting for request plugins to Stop"))
			return errs.ToErr()
		}
	}
	return nil
}

func (r *requestModifier) ModifyRequest(req *shared.Request) (*shared.Request, error) {
	var err error
	for _, pluginManager := range r.pluginManagers {
		plugin, err := pluginManager.Get()
		if err != nil {
			return req, fmt.Errorf("UNABLE_TO_FETCH_REQUEST_MODIFIER")
		}

		requestPlugin, ok := plugin.(request_plugin.PluginRPC)
		if !ok {
			return req, fmt.Errorf("INVALID_REQUEST_PLUGIN_TYPE")
		}

		req, err = requestPlugin.ModifyRequest(req)
		if err != nil {
			break
		}
	}

	return req, err
}

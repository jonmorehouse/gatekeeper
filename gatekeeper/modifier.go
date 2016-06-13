package gatekeeper

import (
	"fmt"
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type Modifier interface {
	Start() error
	Stop(time.Duration) error
	ModifyRequest(*shared.Request) (*shared.Request, error)
	ModifyResponse(*shared.Request, *shared.Response) (*shared.Response, error)
}

type ModifierClient interface {
	ModifyRequest(*shared.Request) (*shared.Request, error)
	ModifyResponse(*shared.Request, *shared.Response) (*shared.Response, error)
}

type modifier struct {
	pluginManagers []PluginManager
}

func NewModifier(pluginManagers []PluginManager) Modifier {
	return &modifier{
		pluginManagers: pluginManagers,
	}
}

func (r *modifier) Start() error {
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

func (r *modifier) Stop(duration time.Duration) error {
	errs := NewAsyncMultiError()

	var wg sync.WaitGroup
	doneCh := make(chan struct{})

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

func (r *modifier) getModifierClient(manager PluginManager) (ModifierClient, error) {
	rawPlugin, err := manager.Get()
	if err != nil {
		return nil, fmt.Errorf("UNABLE_TO_FETCH_MODIFIER_PLUGIN")
	}

	modifier, ok := rawPlugin.(ModifierClient)
	if !ok {
		return nil, fmt.Errorf("UNABLE_TO_CONVERT_PLUGIN_TO_MODIFIER")
	}

	return modifier, nil
}

func (r *modifier) ModifyRequest(req *shared.Request) (*shared.Request, error) {
	var err error
	for _, pluginManager := range r.pluginManagers {
		modifier, err := r.getModifierClient(pluginManager)
		if err != nil {
			return req, err
		}

		req, err = modifier.ModifyRequest(req)
		if err != nil {
			break
		}
	}

	return req, err
}

func (r *modifier) ModifyResponse(req *shared.Request, resp *shared.Response) (*shared.Response, error) {
	var err error

	for _, pluginManager := range r.pluginManagers {
		modifier, err := r.getModifierClient(pluginManager)
		if err != nil {
			return resp, err
		}

		resp, err = modifier.ModifyResponse(req, resp)
		if err != nil {
			break
		}
	}

	return resp, err
}

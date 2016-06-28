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
	ModifyErrorResponse(error, *shared.Request, *shared.Response) (*shared.Response, error)
}

type ModifierClient interface {
	ModifyRequest(*shared.Request) (*shared.Request, error)
	ModifyResponse(*shared.Request, *shared.Response) (*shared.Response, error)
	ModifyErrorResponse(error, *shared.Request, *shared.Response) (*shared.Response, error)
}

type modifier struct {
	metricWriter   MetricWriterClient
	pluginManagers []PluginManager
}

func NewModifier(pluginManagers []PluginManager, metricWriter MetricWriterClient) Modifier {
	return &modifier{
		metricWriter:   metricWriter,
		pluginManagers: pluginManagers,
	}
}

func (r *modifier) Start() error {
	errs := NewMultiError()

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
	errs := NewMultiError()

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
		shared.ProgrammingError("PluginManager has no instances available")
		return nil, InternalPluginError
	}

	modifier, ok := rawPlugin.(ModifierClient)
	if !ok {
		shared.ProgrammingError("PluginManager returned an instance that was not a ModifierClient")
		return nil, InternalPluginError
	}

	return modifier, nil
}

func (r *modifier) ModifyRequest(req *shared.Request) (*shared.Request, error) {
	errs := NewMultiError()

	for _, pluginManager := range r.pluginManagers {
		modifier, err := r.getModifierClient(pluginManager)
		if err != nil {
			errs.Add(err)
			continue
		}

		startTS := time.Now()
		req, err = modifier.ModifyRequest(req)
		pluginManager.WriteMetric("ModifyRequest", time.Now().Sub(startTS), err)
		if err != nil {
			errs.Add(err)
		}
	}

	return req, errs.ToErr()
}

func (r *modifier) ModifyResponse(req *shared.Request, resp *shared.Response) (*shared.Response, error) {
	errs := NewMultiError()

	for _, pluginManager := range r.pluginManagers {
		modifier, err := r.getModifierClient(pluginManager)
		if err != nil {
			errs.Add(err)
			continue
		}

		startTS := time.Now()
		resp, err = modifier.ModifyResponse(req, resp)
		pluginManager.WriteMetric("ModifyResponse", time.Now().Sub(startTS), err)
		if err != nil {
			errs.Add(err)
		}
	}

	return resp, errs.ToErr()
}

func (r *modifier) ModifyErrorResponse(err error, req *shared.Request, resp *shared.Response) (*shared.Response, error) {
	errs := NewMultiError()

	for _, pluginManager := range r.pluginManagers {
		modifier, err := r.getModifierClient(pluginManager)
		if err != nil {
			errs.Add(err)
			continue
		}

		startTS := time.Now()
		resp, err = modifier.ModifyErrorResponse(err, req, resp)
		pluginManager.WriteMetric("ModifyErrorResponse", time.Now().Sub(startTS), err)
		if err != nil {
			errs.Add(err)
		}
	}

	return resp, err
}

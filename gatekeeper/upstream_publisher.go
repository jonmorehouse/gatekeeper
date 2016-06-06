package gatekeeper

import (
	"fmt"
	"sync"
	"time"

	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
	"github.com/jonmorehouse/gatekeeper/shared"
)

// TODO come up with a better sort of Stop/Start interface here?
type Publisher interface {
	Start() error
	Stop(time.Duration) error
}

type UpstreamPublisherAndManager interface {
	Publisher
	upstream_plugin.Manager
}

// UpstreamPublisher starts, maintains and wraps an UpstreamPlugin, accepting
// events from the plugin. Each plugin event is serialized into the correct
// type and published to the broadcaster type to ensure that all listeners
// receive the message correctly.
type UpstreamPublisher struct {
	// pluginManager wraps one or more plugins of the same type and ensures
	// that they survive, correctly. In practice, we will want a count of 1
	// instances for each PluginManager here to avoid duplicates.
	pluginManagers []PluginManager
	broadcaster    EventBroadcaster

	// keep a tally of the upstreams / plugins we've seen here by ID only
	knownUpstreams map[shared.UpstreamID]interface{}
	knownBackends  map[shared.BackendID]interface{}
}

func NewUpstreamPublisher(pluginManagers []PluginManager, broadcaster EventBroadcaster) UpstreamPublisherAndManager {
	return &UpstreamPublisher{
		pluginManagers: pluginManagers,
		broadcaster:    broadcaster,
	}
}

func (p *UpstreamPublisher) Start() error {
	errs := NewAsyncMultiError()

	var wg sync.WaitGroup

	// start all instances of all plugins (which is just 1 instance per
	// unique plugin here)
	for _, manager := range p.pluginManagers {
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

func (p *UpstreamPublisher) Stop(dur time.Duration) error {
	errs := NewAsyncMultiError()
	timeout := time.Now().Add(dur)

	var wg sync.WaitGroup
	doneCh := make(chan interface{})

	// stop all pluginManagers, waiting for each one at the end!
	for _, manager := range p.pluginManagers {
		wg.Add(1)
		go func(p PluginManager) {
			defer wg.Done()
			if err := p.Stop(dur); err != nil {
				errs.Add(err)
			}
		}(manager)
	}

	go func() {
		wg.Wait()
		doneCh <- struct{}{}
	}()

	for {
		select {
		case <-doneCh:
			return errs.ToErr()
		default:
			if time.Now().After(timeout) {
				return errs.ToErr()
			}
		}
	}
}

func (p *UpstreamPublisher) AddUpstream(upstream shared.Upstream) error {
	err := p.broadcaster.Publish(UpstreamEvent{
		EventType:  UpstreamAdded,
		Upstream:   upstream,
		UpstreamID: upstream.ID,
	})
	if err != nil {
		return shared.NewError(fmt.Errorf("UNABLE_TO_BROADCAST_MESSAGE"))
	}
	return nil
}

func (p *UpstreamPublisher) RemoveUpstream(upstreamID shared.UpstreamID) error {
	if _, ok := p.knownUpstreams[upstreamID]; !ok {
		return shared.NewError(fmt.Errorf("UPSTREAM_NOT_FOUND"))
	}

	delete(p.knownUpstreams, upstreamID)
	err := p.broadcaster.Publish(UpstreamEvent{
		EventType:  UpstreamRemoved,
		UpstreamID: upstreamID,
	})
	if err != nil {
		return shared.NewError(fmt.Errorf("UNABLE_TO_BROADCAST_MESSAGE"))
	}
	return nil
}

func (p *UpstreamPublisher) AddBackend(upstreamID shared.UpstreamID, backend shared.Backend) error {
	if _, ok := p.knownUpstreams[upstreamID]; !ok {
		return shared.NewError(fmt.Errorf("UPSTREAM_NOT_FOUND"))
	}

	err := p.broadcaster.Publish(UpstreamEvent{
		EventType:  BackendAdded,
		UpstreamID: upstreamID,
		BackendID:  backend.ID,
		Backend:    backend,
	})
	if err != nil {
		return shared.NewError(fmt.Errorf("UNABLE_TO_BROADCAST_MESSAGE"))
	}
	return nil
}

func (p *UpstreamPublisher) RemoveBackend(backendID shared.BackendID) error {
	if _, ok := p.knownBackends[backendID]; !ok {
		return shared.NewError(fmt.Errorf("BACKEND_NOT_FOUND"))
	}
	err := p.broadcaster.Publish(UpstreamEvent{
		EventType: BackendRemoved,
		BackendID: backendID,
	})
	if err != nil {
		return shared.NewError(fmt.Errorf("UNABLE_TO_BROADCAST_MESSAGE"))
	}
	return nil
}

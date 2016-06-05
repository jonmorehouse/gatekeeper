package gatekeeper

import (
	"fmt"
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

// TODO come up with a better sort of Stop/Start interface here?
type Publisher interface {
	Start() error
	Stop(time.Duration) error
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

func NewUpstreamPublisher(pluginManagers []PluginManager, broadcaster EventBroadcaster) Publisher {
	return &UpstreamPublisher{
		pluginManagers: pluginManagers,
		broadcaster:    broadcaster,
	}
}

func (p *UpstreamPublisher) Start() error {
	errs := NewAsyncMultiError()

	var wg sync.WaitGroup
	defer wg.Wait()

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
	return p.broadcaster.Publish(UpstreamEvent{
		EventType:  UpstreamAdded,
		Upstream:   upstream,
		UpstreamID: upstream.ID,
	})
}

func (p *UpstreamPublisher) RemoveUpstream(upstreamID shared.UpstreamID) error {
	if _, ok := p.knownUpstreams[upstreamID]; !ok {
		return fmt.Errorf("Unknown upstream")
	}

	delete(p.knownUpstreams, upstreamID)
	return p.broadcaster.Publish(UpstreamEvent{
		EventType:  UpstreamRemoved,
		UpstreamID: upstreamID,
	})
}

func (p *UpstreamPublisher) AddBackend(upstreamID shared.UpstreamID, backend shared.Backend) error {
	if _, ok := p.knownUpstreams[upstreamID]; !ok {
		return fmt.Errorf("Unknown upstream")
	}

	return p.broadcaster.Publish(UpstreamEvent{
		EventType:  BackendAdded,
		UpstreamID: upstreamID,
		BackendID:  backend.ID,
		Backend:    backend,
	})
}

func (p *UpstreamPublisher) RemoveBackend(backendID shared.BackendID) error {
	if _, ok := p.knownBackends[backendID]; !ok {
		return fmt.Errorf("Unknown backend")
	}
	return p.broadcaster.Publish(UpstreamEvent{
		EventType: BackendRemoved,
		BackendID: backendID,
	})
}

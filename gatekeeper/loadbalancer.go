package gatekeeper

import (
	"fmt"
	"log"
	"sync"
	"time"

	loadbalancer_plugin "github.com/jonmorehouse/gatekeeper/plugin/loadbalancer"
	"github.com/jonmorehouse/gatekeeper/shared"
)

// LoadBalancerManager is responsible for handling Backend events and piping
// them along to a load balancer. Specifically, this entails listening to a
// Broadcaster for the BackendRemoved and BackendAdded events and from there,
// updating each instance of the plugin to ensure that all backend load
// balancers have all the backends needed
type LoadBalancer interface {
	Start() error
	GetBackend(*shared.Upstream) (shared.Backend, error)
	Stop(time.Duration) error
}

type LoadBalancerClient interface {
	GetBackend(*shared.Upstream) (shared.Backend, error)
}

type loadBalancer struct {
	// dependencies
	broadcaster   EventBroadcaster
	pluginManager PluginManager

	// internal
	eventCh  EventCh
	listenID EventListenerID
	stopCh   chan interface{}
}

func NewLoadBalancer(broadcaster EventBroadcaster, pluginManager PluginManager) LoadBalancer {
	return &loadBalancer{
		broadcaster:   broadcaster,
		pluginManager: pluginManager,
		eventCh:       make(EventCh),
		stopCh:        make(chan interface{}),
	}
}

func (l *loadBalancer) Start() error {
	// start the loadBalancer plugins
	if err := l.pluginManager.Start(); err != nil {
		return err
	}

	// configure this object to receive the correct events from the
	// EventBroadcaster and process them correctly.
	listenID, err := l.broadcaster.AddListener(l.eventCh, []EventType{BackendAdded, BackendRemoved})
	if err != nil {
		return err
	}
	l.listenID = listenID
	go l.worker()
	return nil
}

func (l *loadBalancer) Stop(duration time.Duration) error {
	timeout := time.Now().Add(duration)
	errs := NewAsyncMultiError()
	if err := l.broadcaster.RemoveListener(l.listenID); err != nil {
		errs.Add(err)
	}

	doneCh := make(chan interface{})
	var wg sync.WaitGroup
	wg.Add(2)

	// stop all of the plugins in this pluginManager
	go func() {
		if err := l.pluginManager.Stop(duration); err != nil {
			errs.Add(err)
		}
		wg.Done()
	}()

	// wait for the internal worker to stop
	go func() {
		l.stopCh <- struct{}{}
		<-l.stopCh
		wg.Done()
	}()

	// wait for the waitGroup to be finished
	go func() {
		wg.Wait()
		doneCh <- struct{}{}
	}()

	// now wait until this entire method stops or times out
	for {
		select {
		case <-doneCh:
			goto done
		default:
			if time.Now().After(timeout) {
				errs.Add(fmt.Errorf("Did not stop quickly enough"))
				goto done
			}
		}
	}
done:
	return errs.ToErr()
}

func (l *loadBalancer) worker() {
	for {
		select {
		case event := <-l.eventCh:
			upstreamEvent, ok := event.(UpstreamEvent)
			if !ok {
				log.Fatal("Invalid event was broadcast")
			}
			go l.handleEvent(upstreamEvent)
		case <-l.stopCh:
			l.stopCh <- struct{}{}
			return
		}
	}
}

func (l *loadBalancer) handleEvent(event UpstreamEvent) {
	var cb func(loadbalancer_plugin.Plugin) error

	switch event.EventType {
	case BackendAdded:
		cb = func(p loadbalancer_plugin.Plugin) error {
			return p.AddBackend(event.UpstreamID, event.Backend)
		}
	case BackendRemoved:
		cb = func(p loadbalancer_plugin.Plugin) error {
			return p.RemoveBackend(event.Backend)
		}
	default:
		log.Fatal("Broadcaster sent an unsubscribed for event")
	}

	plugins, err := l.pluginManager.All()
	if err != nil {
		log.Println(err)
	}

	var wg sync.WaitGroup
	errs := NewAsyncMultiError()
	for _, plugin := range plugins {
		wg.Add(1)

		go func(p loadbalancer_plugin.Plugin) {
			defer wg.Done()
			if err := Retry(func() error { return cb(p) }, 3); err != nil {
				errs.Add(err)
			}
		}(plugin.(loadbalancer_plugin.Plugin))
	}
}

func (l *loadBalancer) GetBackend(upstream *shared.Upstream) (shared.Backend, error) {
	plugin, err := l.pluginManager.Get()
	if err != nil {
		return shared.NilBackend, err
	}

	// NOTE we only pass along the UpstreamID because we don't want to send
	// the entirety of the upstream over the wire each and every time.
	return plugin.(loadbalancer_plugin.Plugin).GetBackend(upstream.ID)
}

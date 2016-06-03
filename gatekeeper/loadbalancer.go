package gatekeeper

import (
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jonmorehouse/gatekeeper/shared"
)

// LoadBalancerManager is responsible for handling Backend events and piping
// them along to a load balancer. Specifically, this entails listening to a
// Broadcaster for the BackendRemoved and BackendAdded events and from there,
// updating each instance of the plugin to ensure that all backend load
// balancers have all the backends needed
type LoadBalancerManager interface {
	Start() error
	Stop(time.Duration) error
}

type LoadBalancerClient interface {
	GetBackend(shared.UpstreamID) shared.Backend
}

type LoadBalancer struct {
	// dependencies
	broadcaster   Broadcaster
	pluginManager PluginManager

	// internal
	eventCh  EventCh
	listenID ListenID
	stopCh   chan interface{}
}

func NewLoadBalancer(broadcaster EventBroadcaster, pluginManager PluginManager) *LoadBalancer {
	return &LoadBalancer{
		broadcaster:   broadcaster,
		pluginManager: pluginManager,
		listenCh:      make(ListenCh),
		stopCh:        make(chan interface{}),
	}
}

func (l *LoadBalancer) Start() error {
	listenID, err := l.broadcaster.AddListener(l.eventCh, []EventType{BackendAdded, BackendRemoved})
	if err != nil {
		return err
	}

	l.listenID = listenID
	go l.worker()
	return nil
}

func (l *LoadBalancer) Stop(duration time.Duration) error {
	timeout := time.Now().Add(duration)
	errs := NewAsyncMultiError()
	if err := l.RemoveListener(l.listenID); err != nil {
		errs.Add(err)
	}

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
			break
		case time.Now().After(timeout):
			errs.Add(fmt.Errorf("Did not stop quickly enough"))
		default:
		}
	}
	return nil
}

func (l *LoadBalancer) worker() {
	for {
		select {
		case event := <-l.eventCh:
			upstreamEvent, ok := event.(UpstreamEvent)
			if !ok {
				log.Fatal("Invalid event was broadcast")
			}
			go l.handleEvent(upstreamEvent)
		case <-l.stopCh:
			l.stop <- struct{}{}
			return
		}
	}
}

func (l *LoadBalancer) handleEvent(event UpstreamEvent) {
	var cb func(loadbalancer.Plugin) error

	switch event.EventType {
	case BackendAdded:
		cb = func(p Plugin) error {
			return p.AddBackend(event.UpstreamID, event.Backend)
		}
	case BackendRemoved:
		cb = func(p Plugin) error {
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
		go func(p loadbalancer.Plugin) {
			defer wg.Done()
			if err := Retry(cb(p), 3); err != nil {
				errs.Add(err)
			}
		}(plugin)
	}

	// TODO handle errors here ...
}

func (l *LoadBalancer) GetBackend(upstream shared.Upstream) (shared.Backend, error) {
	plugin, err := l.pluginManager.Get()
	if err != nil {
		return shared.NilBackend, err
	}

	// NOTE we only pass along the UpstreamID because we don't want to send
	// the entirety of the upstream over the wire each and every time.
	return plugin.GetBackend(upstream.ID)
}

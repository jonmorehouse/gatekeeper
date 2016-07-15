package core

import (
	"log"
	"sync"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	loadbalancer_plugin "github.com/jonmorehouse/gatekeeper/plugin/loadbalancer"
)

type LoadBalancerClient interface {
	GetBackend(gatekeeper.UpstreamID) (*gatekeeper.Backend, error)
}

type LoadBalancer interface {
	startStopper

	LoadBalancerClient
}

func NewLocalLoadBalancer(broadcaster Broadcaster) LoadBalancer {
	return &localLoadBalancer{
		backends:   make(map[gatekeeper.UpstreamID]map[gatekeeper.BackendID]*gatekeeper.Backend),
		Subscriber: NewSubscriber(broadcaster),
	}
}

type localLoadBalancer struct {
	backends map[gatekeeper.UpstreamID]map[gatekeeper.BackendID]*gatekeeper.Backend

	Subscriber

	sync.RWMutex
}

func (l *localLoadBalancer) Start() error {
	l.AddUpstreamEventHook(gatekeeper.BackendAddedEvent, l.addBackendHook)
	l.AddUpstreamEventHook(gatekeeper.BackendRemovedEvent, l.removeBackendHook)
	return l.Subscriber.Start()
}

func (l *localLoadBalancer) GetBackend(upstreamID gatekeeper.UpstreamID) (*gatekeeper.Backend, error) {
	l.RLock()
	defer l.RUnlock()

	upstreamBackends, found := l.backends[upstreamID]
	if !found {
		return nil, BackendNotFoundError
	}

	// because the go runtime provides randomization out of the box, we can
	// just iterate until we find an element.
	for _, backend := range upstreamBackends {
		return backend, nil
	}

	return nil, BackendNotFoundError
}

func (l *localLoadBalancer) addBackendHook(event *UpstreamEvent) {
	l.Lock()
	defer l.Unlock()

	_, ok := l.backends[event.UpstreamID]
	if !ok {
		l.backends[event.UpstreamID] = make(map[gatekeeper.BackendID]*gatekeeper.Backend)
	}

	l.backends[event.UpstreamID][event.BackendID] = event.Backend
}

func (l *localLoadBalancer) removeBackendHook(event *UpstreamEvent) {
	l.Lock()
	defer l.Unlock()

	_, ok := l.backends[event.UpstreamID]
	if !ok {
		return
	}

	delete(l.backends[event.UpstreamID], event.BackendID)
}

func NewPluginLoadBalancer(broadcaster Broadcaster, pluginManager PluginManager) LoadBalancer {
	return &pluginLoadBalancer{
		pluginManager: pluginManager,
		Subscriber:    NewSubscriber(broadcaster),
	}
}

// pluginLoadBalancer accepts a loadBalancer plugin and a subscriber and is
// responsible for passing along upstream events to the loadbalancer.
// Specifically, it uses the subscriber to configure hooks to call into the
// plugin when a new event is emitted internally
type pluginLoadBalancer struct {
	pluginManager PluginManager
	Subscriber
}

func (l *pluginLoadBalancer) Start() error {
	l.AddUpstreamEventHook(gatekeeper.BackendAddedEvent, l.addBackendHook)
	l.AddUpstreamEventHook(gatekeeper.BackendRemovedEvent, l.removeBackendHook)
	return l.Subscriber.Start()
}

func (l *pluginLoadBalancer) GetBackend(upstreamID gatekeeper.UpstreamID) (*gatekeeper.Backend, error) {
	var backend *gatekeeper.Backend
	var err error

	l.pluginManager.Call("GetBackend", func(plugin Plugin) error {
		lbPlugin, ok := plugin.(loadbalancer_plugin.PluginClient)
		if !ok {
			err = InternalPluginError
			return nil
		}

		backend, err = lbPlugin.GetBackend(upstreamID)
		return err
	})

	return backend, err
}

func (l *pluginLoadBalancer) addBackendHook(event *UpstreamEvent) {
	log.Println("add backend call")
	l.pluginManager.Call("AddBackend", func(plugin Plugin) error {
		lbPlugin, ok := plugin.(loadbalancer_plugin.PluginClient)
		if !ok {
			log.Println(InternalPluginError)
			return nil
		}

		if err := lbPlugin.AddBackend(event.UpstreamID, event.Backend); err != nil {
			log.Println(err)
			return err
		}

		return nil
	})
}

func (l *pluginLoadBalancer) removeBackendHook(event *UpstreamEvent) {
	log.Println("remove backend call")
	l.pluginManager.Call("RemoveBackend", func(plugin Plugin) error {
		lbPlugin, ok := plugin.(loadbalancer_plugin.PluginClient)
		if !ok {
			log.Println(InternalPluginError)
			return nil
		}

		if err := lbPlugin.RemoveBackend(event.Backend); err != nil {
			log.Println(err)
			return err
		}

		return nil
	})
}

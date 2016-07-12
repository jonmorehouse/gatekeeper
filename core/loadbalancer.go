package core

import (
	"log"
	"sync"
)

type LoadBalancer interface {
	StartStopper

	GetBackend(gatekeeper.UpstreamID) (*gatekeeper.Backend, error)
}

func NewLocalLoadBalancer(broadcaster Broadcaster) LoadBalancer {
	return &localLoadBalancer{
		subscriber: NewSubscriber(broadcaster),
	}
}

type localLoadBalancer struct {
	backends map[gatekeeper.UpstreamID]map[gatekeeper.BackendID]*gatekeeper.Backend

	sync.RWMutex
}

func (l *localLoadBalancer) Start() error {
	l.subscriber.AddUpstreamEventhook(gatekeeper.BackendAddedEvent, l.addBackendHook)
	l.subscriber.AddUpstreamEventhook(gatekeeper.BackendRemovedEvent, l.removeBackendHook)
	return l.subscriber.Start()
}

func (l *localLoadBalancer) GetBackend(upstreamID *gatekeeper.UpstreamID) (*gatekeeper.Backend, error) {
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
		subscriber:    NewSubscriber(broadcaster),
		pluginManager: pluginManager,
	}
}

// pluginLoadBalancer accepts a loadBalancer plugin and a subscriber and is
// responsible for passing along upstream events to the loadbalancer.
// Specifically, it uses the subscriber to configure hooks to call into the
// plugin when a new event is emitted internally
type pluginLoadBalancer struct {
	subscriber    Subscriber
	pluginManager PluginManager
}

func (l *pluginLoadBalancer) Start() error {
	l.subscriber.AddHook(gatekeeper.BackendAddedEvent, l.backendAddedHook)
	l.subscriber.AddHook(gatekeeper.BackendRemovedEvent, l.backendRemovedHook)
	return l.subscriber.Start()
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

func (l *pluginLoadBalancer) backendAddedEvent(event *UpstreamEvent) {
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

func (l *pluginLoadBalancer) backendRemovedHook(event *UpstreamEvent) {
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

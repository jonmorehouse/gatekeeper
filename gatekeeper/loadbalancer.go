package gatekeeper

import (
	"log"
	"sync"
)

type LoadBalancer interface {
	StartStopper

	GetBackend(shared.UpstreamID) (*shared.Backend, error)
}

func NewLocalLoadBalancer(broadcaster Broadcaster) LoadBalancer {
	return &localLoadBalancer{
		subscriber: NewSubscriber(broadcaster),
	}
}

type localLoadBalancer struct {
	backends map[shared.UpstreamID]map[shared.BackendID]*shared.Backend

	sync.RWMutex
}

func (l *localLoadBalancer) Start() error {
	l.subscriber.AddUpstreamEventhook(shared.BackendAddedEvent, l.addBackendHook)
	l.subscriber.AddUpstreamEventhook(shared.BackendRemovedEvent, l.removeBackendHook)
	return l.subscriber.Start()
}

func (l *localLoadBalancer) GetBackend(upstreamID *shared.UpstreamID) (*shared.Backend, error) {
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
		l.backends[event.UpstreamID] = make(map[shared.BackendID]*shared.Backend)
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
	l.subscriber.AddHook(shared.BackendAddedEvent, l.backendAddedHook)
	l.subscriber.AddHook(shared.BackendRemovedEvent, l.backendRemovedHook)
	return l.subscriber.Start()
}

func (l *pluginLoadBalancer) GetBackend(upstreamID shared.UpstreamID) (*shared.Backend, error) {
	var backend *shared.Backend
	var err error

	l.pluginManager.Call("GetBackend", func(plugin Plugin) {
		lbPlugin, ok := plugin.(loadbalancer_plugin.Plugin)
		if !ok {
			err = InternalPluginError
			return
		}

		backend, err = lbPlugin.GetBackend(upstreamID)
	})

	return backend, err
}

func (l *pluginLoadBalancer) backendAddedEvent(event *UpstreamEvent) {
	l.pluginManager.Call("AddBackend", func(plugin Plugin) {
		lbPlugin, ok := plugin.(loadbalancer_plugin.Plugin)
		if !ok {
			log.Println(InternalPluginError)
			return
		}

		if err := lbPlugin.AddBackend(event.UpstreamID, event.Backend); err != nil {
			log.Println(err)
		}
	})
}

func (l *pluginLoadBalancer) backendRemovedHook(event *UpstreamEvent) {
	l.pluginManager.Call("RemoveBackend", func(plugin Plugin) {
		lbPlugin, ok := plugin.(loadbalancer_plugin.Plugin)
		if !ok {
			log.Println(InternalPluginError)
			return
		}

		if err := lbPlugin.RemoveBackend(event.Backend); err != nil {
			log.Println(err)
		}
	})
}

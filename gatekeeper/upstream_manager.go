package gatekeeper

import (
	"fmt"
	"log"
	"sync"

	"github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type InternalUpstreamManager interface {
	// start is used by the upstreamDirector to start a _connected_
	// plugin's lifecycle. In most cases, this involves making an RPC call
	// to the plugin and starting the "event flow" of upstream/backend info
	// from the plugin to this struct.
	Start() error
	Stop() error
}

// UpstreamManager implements the upstream.Manager interface and as such, is
// used as the "backend" behind callbacks from children plugins.
type UpstreamManager struct {
	plugin upstream.Plugin

	// internal store for storing upstreams and plugins
	upstreams        map[upstream.UpstreamID]upstream.Upstream
	upstreamBackends map[upstream.UpstreamID][]upstream.Backend

	// we wrap a rwmutex around the internal state modifiers in this struct
	// so as to handle multithreaded access from our plugin. In practice,
	// this might not be necessary
	sync.RWMutex
}

func NewUpstreamManager(pluginOpts PluginOpts) (InternalUpstreamManager, error) {
	manager := UpstreamManager{
		upstreams:        make(map[upstream.UpstreamID]upstream.Upstream),
		upstreamBackends: make(map[upstream.UpstreamID][]upstream.Backend),
	}

	pluginClient, err := upstream.NewClient(pluginOpts.Name, pluginOpts.Cmd)
	if err != nil {
		return nil, err
	}

	if err := pluginClient.Configure(pluginOpts.Opts); err != nil {
		return nil, err
	}

	manager.plugin = pluginClient
	return &manager, nil
}

func (m *UpstreamManager) Start() error {
	if err := m.plugin.Start(m); err != nil {
		return err
	}
	return nil
}

func (m *UpstreamManager) Stop() error {
	if err := m.plugin.Stop(); err != nil {
		return err
	}
	return nil
}

func (m *UpstreamManager) AddUpstream(u upstream.Upstream) (upstream.UpstreamID, error) {
	// This method is called from upstream plugins and specifically is
	// blocking on their end as tehy wait for a response from the RPC
	// Client. This method will update the internal store and will ensure
	// that we have returned the correct upstreamID or an error if the
	// operation is invalid.
	m.Lock()
	defer m.Unlock()

	if u.ID == upstream.NilUpstreamID {
		u.ID = NewUpstreamID()
	}

	m.upstreams[u.ID] = u
	return u.ID, nil
}

func (m *UpstreamManager) RemoveUpstream(uID upstream.UpstreamID) error {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.upstreams[uID]; !ok {
		return fmt.Errorf("Upstream does not exist")
	}

	delete(m.upstreams, uID)
	return nil
}

func (m *UpstreamManager) AddBackend(uID upstream.UpstreamID, backend upstream.Backend) (upstream.BackendID, error) {
	m.Lock()
	defer m.Unlock()

	if _, ok := m.upstreams[uID]; !ok {
		return upstream.NilBackendID, fmt.Errorf("Upstream does not exist. Please add upstream before adding backends...")
	}

	backend.ID = NewBackendID()
	if _, ok := m.upstreamBackends[uID]; !ok {
		m.upstreamBackends[uID] = make([]upstream.Backend, 0)
	}
	m.upstreamBackends[uID] = append(m.upstreamBackends[uID], backend)
	return backend.ID, nil
}

func (m *UpstreamManager) RemoveBackend(bID upstream.BackendID) error {
	m.Lock()
	defer m.Unlock()

	// meh I didn't design this very well ...
	// maybe just use an associative array with uID / backendIDs instead?
	for upstreamID, upstreamBackends := range m.upstreamBackends {
		for index, backend := range upstreamBackends {
			if backend.ID != bID {
				continue
			}
			// remove this backend from the list of backends for this particular upstream.
			m.upstreamBackends[upstreamID] = append(upstreamBackends[:index], upstreamBackends[index+1:]...)
			return nil
		}
	}

	return fmt.Errorf("Backend not found")
}

func (m *UpstreamManager) TempFetch() (upstream.Upstream, upstream.Backend) {
	log.Println("This is temporary until we build out Queryier interface ...")
	m.RLock()
	defer m.RUnlock()

	for upstreamID, upstr := range m.upstreams {
		return upstr, m.upstreamBackends[upstreamID][0]
	}
	return upstream.NilUpstream, upstream.NilBackend
}

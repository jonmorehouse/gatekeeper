package main

import (
	"log"
	"sync"
	"time"

	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type ManagerError uint

var managerErrorMapping = map[ManagerError]string{
	UpstreamDatabaseError:  "upstream database error",
	UpstreamRPCError:       "upstream rpc error",
	UpstreamDuplicateError: "duplicate upstream",
	UpstreamNotFoundError:  "upstream not found",

	BackendDatabaseError:  "backend database error",
	BackendRPCError:       "backend rpc error",
	BackendDuplicateError: "duplicate backend error",
	BackendNotFoundError:  "backend not found error",

	InternalError: "internal error",
}

const (
	UpstreamDatabaseError ManagerError = iota + 1
	UpstreamRPCError
	UpstreamDuplicateError
	UpstreamNotFoundError

	BackendDatabaseError
	BackendRPCError
	BackendDuplicateError
	BackendNotFoundError

	InternalError
)

func (m ManagerError) Error() string {
	str, ok := managerErrorMapping[m]
	if !ok {
		log.Fatal("programming error: unknown error type")
	}

	return str
}

type Manager interface {
	Start() error
	Stop() error

	AddUpstream(*gatekeeper.Upstream) (gatekeeper.UpstreamID, error)
	RemoveUpstream(gatekeeper.UpstreamID) error

	AddBackend(gatekeeper.UpstreamID, *gatekeeper.Backend) (gatekeeper.BackendID, error)
	RemoveBackend(gatekeeper.BackendID) error
}

type ManagerClient interface {
	AddUpstream(*gatekeeper.Upstream) (gatekeeper.UpstreamID, error)
	RemoveUpstream(gatekeeper.UpstreamID) error

	AddBackend(gatekeeper.UpstreamID, *gatekeeper.Backend) (gatekeeper.BackendID, error)
	RemoveBackend(gatekeeper.BackendID) error
}

type manager struct {
	config     *Config
	database   DatabaseClient
	rpcManager upstream_plugin.Manager

	stopCh    chan struct{} // signify the worker to stop
	stoppedCh chan struct{} // signify that the worker has stopped
	skipCh    chan struct{} // signify the worker to skip a sync operation
	errCh     chan error

	// how often we sync with MySQL and update the parent process
	syncInterval time.Duration

	*sync.RWMutex

	// internal storage, which is mutex protected
	upstreams        map[gatekeeper.UpstreamID]*gatekeeper.Upstream
	backends         map[gatekeeper.BackendID]*gatekeeper.Backend
	upstreamBackends map[gatekeeper.UpstreamID]map[gatekeeper.BackendID]struct{}
}

func NewManager(config *Config, database Database, rpcManager upstream_plugin.Manager) Manager {
	return &manager{
		config:     config,
		database:   database,
		rpcManager: rpcManager,

		// attrs for managing the run-loop
		stopCh:       make(chan struct{}), // signify the worker to stop
		stoppedCh:    make(chan struct{}), // signify that the worker has stopped
		skipCh:       make(chan struct{}), // skip a sync
		syncInterval: time.Millisecond * 500,

		upstreams:        make(map[gatekeeper.UpstreamID]*gatekeeper.Upstream),
		backends:         make(map[gatekeeper.BackendID]*gatekeeper.Backend),
		upstreamBackends: make(map[gatekeeper.UpstreamID]map[gatekeeper.BackendID]struct{}),
	}
}

// sync the manager with the database, and start the background process to keep the service in sync
func (m *manager) Start() error {
	if err := m.sync(); err != nil {
		return err
	}

	go m.worker()
	return nil
}

// stop the worker
func (m *manager) Stop() error {
	m.stopCh <- struct{}{}
	<-m.stoppedCh
	return nil
}

// add and upstream to the database and local gatekeeper processes, returning a unique global ID
func (m *manager) AddUpstream(newUpstream *gatekeeper.Upstream) (gatekeeper.UpstreamID, error) {
	// tell the background worker to skip a sync because we want to make
	// sure we have the best representation of the data possible here.
	m.skipCh <- struct{}{}
	m.sync()

	if newUpstream.ID != gatekeeper.NilUpstreamID {
		// we don't support update for upstreams
		return gatekeeper.NilUpstreamID, UpstreamDuplicateError
	}

	upstreamID, err := m.database.AddUpstream(newUpstream)
	if err != nil {
		log.Println("unable to add upstream to database: ", err)
		return gatekeeper.NilUpstreamID, UpstreamDatabaseError
	}

	newUpstream.ID = upstreamID
	return upstreamID, m.addLocalUpstream(newUpstream)
}

// remove an upstream and all of its backends
func (m *manager) RemoveUpstream(upstreamID gatekeeper.UpstreamID) error {
	// reset the interval to tell the background worker to skip a sync,
	// since we're doing it here and want to make sure we have the latest
	// data
	m.skipCh <- struct{}{}
	m.sync()

	if err := m.database.RemoveUpstream(upstreamID); err != nil {
		log.Println("unable to remove upstream from database: ", err)
		return UpstreamDatabaseError
	}

	// TODO remove backends

	return m.removeLocalUpstream(upstreamID)
}

// add a backend to both the database and the local gatekeeper processes. Return a unique, global ID for the Backend
func (m *manager) AddBackend(upstreamID gatekeeper.UpstreamID, backend *gatekeeper.Backend) (gatekeeper.BackendID, error) {
	// skip a background sync job and sync the data immediately
	m.skipCh <- struct{}{}
	m.sync()

	// backends cannot be updated; any backend without a nil ID will be rejected
	if backend.ID != gatekeeper.NilBackendID {
		return gatekeeper.NilBackendID, BackendDuplicateError
	}

	backendID, err := m.database.AddBackend(upstreamID, backend)
	if err != nil {
		log.Println("unable to add backend to database: ", err)
		return gatekeeper.NilBackendID, BackendDatabaseError
	}

	backend.ID = backendID
	return backendID, m.addLocalBackend(upstreamID, backend)
}

// remove a backend from the Database and the local gatekeeper processes
func (m *manager) RemoveBackend(backendID gatekeeper.BackendID) error {
	m.skipCh <- struct{}{}
	m.sync()

	if err := m.database.RemoveBackend(backendID); err != nil {
		log.Println("unable to remove backend from database: ", err)
		return BackendDatabaseError
	}

	return m.removeLocalBackend(backendID)
}

// ensure that the manager is synced with the database at most after m.syncInterval
func (m *manager) worker() {
	// we track the last error only. Could probably use some more
	// robustness here in future iterations
	for {
		select {
		case <-m.stopCh:
			goto stopped
		case <-m.skipCh:
			continue
		case <-time.After(m.syncInterval):
			if err := m.sync(); err != nil {
				log.Println("unable to sync manager: ", err)
			}
		}
	}
stopped:
	m.stoppedCh <- struct{}{}
}

// sync the local plugin and its parent process with the database; reporting
// any extenuating errors that happen at the database level.
func (m *manager) sync() error {
	// fetch all upstreams from the database
	upstreams, err := m.database.FetchUpstreams()
	if err != nil {
		log.Println("unable to fetch upstreams from database: ", err)
		return UpstreamDatabaseError
	}

	// maintain a list of errors to return back to the callee
	errs := NewMultiError()

	// we maintain a list of state that needs to stick around
	upstreamsToKeep := make(map[gatekeeper.UpstreamID]struct{})
	backendsToKeep := make(map[gatekeeper.BackendID]struct{})

	// for each upstream, sync the local state with the database state if necessary
	for _, upstream := range upstreams {
		backends, err := m.database.FetchUpstreamBackends(upstream.ID)
		if err != nil {
			log.Println("Unable to fetch upstream backends")
			errs.Add(BackendDatabaseError)
		}

		// if the upstream is not yet known locally, then add it
		m.RLock()
		_, ok := m.upstreams[upstream.ID]
		m.RUnlock()
		if !ok {
			if err := m.addLocalUpstream(upstream); err != nil {
				errs.Add(err)
			}
		}

		upstreamsToKeep[upstream.ID] = struct{}{}

		if _, ok := m.upstreamBackends[upstream.ID]; !ok {
			// NOTE this should never be the case due to the fact
			// that we set the upstreamBackends map in the
			// localAddUpstream method.
			m.upstreamBackends[upstream.ID] = make(map[gatekeeper.BackendID]struct{})
			continue
		}

		for _, backend := range backends {
			backendsToKeep[backend.ID] = struct{}{}

			// if the backend already exists, then noop
			if _, ok := m.upstreamBackends[upstream.ID]; ok {
				continue
			}

			if err := m.addLocalBackend(upstream.ID, backend); err != nil {
				errs.Add(err)
			}
		}
	}

	// loop through all known backends and upstreams and decipher which ones need to be deleted
	backendsToRemove := make([]gatekeeper.BackendID, 0, 0)
	upstreamsToRemove := make([]gatekeeper.UpstreamID, 0, 0)

	m.RLock()
	for upstreamID, _ := range m.upstreams {
		if _, ok := upstreamsToKeep[upstreamID]; !ok {
			upstreamsToRemove = append(upstreamsToRemove, upstreamID)
		}
	}
	for backendID, _ := range m.backends {
		if _, ok := backendsToKeep[backendID]; !ok {
			backendsToRemove = append(backendsToRemove, backendID)
		}
	}
	m.RUnlock()

	for _, backendID := range backendsToRemove {
		if err := m.removeLocalBackend(backendID); err != nil {
			errs.Add(err)
		}
	}

	for _, upstreamID := range upstreamsToRemove {
		if err := m.removeLocalUpstream(upstreamID); err != nil {
			errs.Add(err)
		}
	}

	return errs.AsErr()
}

// add a local upstream to the plugin's state, as well as the parent process
func (m *manager) addLocalUpstream(upstream *gatekeeper.Upstream) error {
	m.RLock()
	if _, found := m.upstreams[upstream.ID]; found {
		m.RUnlock()
		return UpstreamDuplicateError
	}
	m.RUnlock()

	// emit the upstream to the parent process over RPC
	if err := m.rpcManager.AddUpstream(upstream); err != nil {
		log.Println("unable to add upstream over RPC: ", err)
		return UpstreamRPCError
	}

	m.Lock()
	defer m.Unlock()

	m.upstreams[upstream.ID] = upstream
	m.upstreamBackends[upstream.ID] = make(map[gatekeeper.BackendID]struct{})
	return nil
}

// remove an upstream and all of its backends from the local plugin state and the parent process
func (m *manager) removeLocalUpstream(upstreamID gatekeeper.UpstreamID) error {
	m.RLock()
	if _, found := m.upstreams[upstreamID]; !found {
		m.RUnlock()
		return UpstreamNotFoundError
	}

	if _, found := m.upstreamBackends[upstreamID]; !found {
		m.RUnlock()
		return UpstreamNotFoundError
	}

	var err error
	for backendID, _ := range m.upstreamBackends[upstreamID] {
		if backendErr := m.removeLocalBackend(backendID); err != nil {
			err = backendErr
		}
	}
	m.RUnlock()

	if rpcErr := m.rpcManager.RemoveUpstream(upstreamID); rpcErr != nil {
		log.Println("unable to remove upstream over RPC: ", rpcErr)
		return UpstreamRPCError
	}
	return err
}

// add a backend to the local cache as well as the parent process
func (m *manager) addLocalBackend(upstreamID gatekeeper.UpstreamID, backend *gatekeeper.Backend) error {
	m.RLock()

	// make sure that the upstream this backend belongs to exists
	_, upstreamFound := m.upstreams[upstreamID]
	_, upstreamBackendsFound := m.upstreamBackends[upstreamID]
	if !upstreamFound || !upstreamBackendsFound {
		m.RUnlock()
		return UpstreamNotFoundError
	}

	// make sure that the backend isn't a duplicate
	_, backendFound := m.backends[backend.ID]
	_, upstreamBackendFound := m.upstreamBackends[upstreamID][backend.ID]
	if backendFound || upstreamBackendFound {
		m.RUnlock()
		return BackendDuplicateError
	}
	m.RUnlock()

	if err := m.rpcManager.AddBackend(upstreamID, backend); err != nil {
		log.Println("unable to add backend over RPC: ", err)
		return BackendRPCError
	}

	// add the backend to the local in memory storage
	m.Lock()
	defer m.Unlock()

	m.backends[backend.ID] = backend
	m.upstreamBackends[upstreamID][backend.ID] = struct{}{}
	return nil
}

// remove the backend from the local state, updating the parent over RPC to remove the backend
func (m *manager) removeLocalBackend(backendID gatekeeper.BackendID) error {
	var err error
	m.Lock()

	// remove the backend
	if _, found := m.backends[backendID]; !found {
		err = BackendNotFoundError
	}
	delete(m.backends, backendID)

	// look through each of the known upstreams and remove the upstream's link to this backend
	found := false
	for upstreamID, upstreamBackends := range m.upstreamBackends {
		if _, ok := upstreamBackends[backendID]; !ok {
			continue
		}
		found = true
		delete(m.upstreamBackends[upstreamID], backendID)
	}
	if !found {
		err = UpstreamNotFoundError
	}
	m.Unlock()

	// remove the backend over RPC
	if rpcErr := m.rpcManager.RemoveBackend(backendID); rpcErr != nil {
		log.Println("unable to remove backend over RPC: ", rpcErr)
		err = BackendRPCError
	}
	return err
}

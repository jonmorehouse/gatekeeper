package main

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

var NoManagerErr = errors.New("No manager set")

type MultiError interface {
	Add(error)
	AsErr() error
}

// multiError is a coroutine safe implementation of the MultiError interface
type multiError struct {
	errs []error
	*sync.RWMutex
}

func NewMultiError() MultiError {
	return &multiError{
		errs: make([]error, 0, 0),
	}
}

func (m *multiError) Add(err error) {
	m.Lock()
	defer m.Unlock()
	m.errs = append(m.errs, err)
}

func (m *multiError) AsErr() error {
	m.RLock()
	defer m.RUnlock()
	if len(m.errs) == 0 {
		return nil
	}

	return m
}

func (m *multiError) Error() string {
	m.RLock()
	defer m.RUnlock()

	errStr := ""
	for _, err := range m.errs {
		errStr = fmt.Sprintf("%s", err.Error())
	}
	return errStr
}

type Config struct {
	Port int

	// datasource name for connecting to the MySQL database,
	// [username[:password]@][protocol[(address)]]/dbname[?param1=value1&...&paramN=valueN]
	DataSourceName string

	// how long we go between checking the the database and our current local state
	SyncInterval time.Duration
}

type Plugin struct {
	config     *Config
	database   Database
	manager    Manager
	rpcManager upstream_plugin.Manager
	api        API

	apiUpstream *gatekeeper.Upstream
	apiBackend  *gatekeeper.Backend
}

func (p *Plugin) Start() error {
	if p.rpcManager == nil {
		return NoManagerErr
	}

	database := NewDatabase()
	if err := database.Connect(p.config.DataSourceName); err != nil {
		return err
	}
	p.database = database

	// build out the manager now, this is responsible for handling the
	// business logic around how we emit upstreams and backends back up to
	// the parent with our granted RPC library
	p.manager = NewManager(p.config, database, p.rpcManager)
	if err := p.manager.Start(); err != nil {
		return err
	}

	// build the API and start it, returning an error if the server failed
	// to start. Furthermore, this
	p.api = NewAPI(p.config, p.manager)
	if err := p.api.Start(); err != nil {
		return err
	}

	// once the server has been started, register this plugin as an
	// upstream and backend itself so this can receive traffic through the
	// gatekeeper proxy itself
	p.apiUpstream = &gatekeeper.Upstream{
		ID:        gatekeeper.NewUpstreamID(),
		Name:      "mysql-upstreams-api",
		Protocols: []gatekeeper.Protocol{gatekeeper.HTTPPublic, gatekeeper.HTTPInternal},
		Hostnames: []string{},
		Prefixes:  []string{"upstreams-plugin"},
		Timeout:   time.Second * 5,
	}

	if err := p.rpcManager.AddUpstream(p.apiUpstream); err != nil {
		return fmt.Errorf("Unable to register plugin as an upstream itself")
	}

	p.apiBackend = &gatekeeper.Backend{
		ID:          gatekeeper.NewBackendID(),
		Address:     fmt.Sprintf("http://127.0.0.1:%d", p.config.Port),
		Healthcheck: "/health",
	}
	if err := p.rpcManager.AddBackend(p.apiUpstream.ID, p.apiBackend); err != nil {
		return fmt.Errorf("Unable to register backend for upstream plugin")
	}

	log.Println("mysql-api-upstreams plugin started...")
	return nil
}

func (p *Plugin) Configure(opts map[string]interface{}) error {
	// parse configuration
	rawPort, ok := opts["listen-port"]
	if !ok {
		return fmt.Errorf("listen-port is required")
	}
	port, ok := rawPort.(float64)
	if !ok {
		return fmt.Errorf("invalid listen-port type")
	}
	p.config.Port = int(port)

	rawDataSourceName, ok := opts["data-source-name"]
	if !ok {
		return fmt.Errorf("data-source-name is required")
	}
	dataSourceName, ok := rawDataSourceName.(string)
	if !ok {
		return fmt.Errorf("invalid data-source-name")
	}
	p.config.DataSourceName = dataSourceName

	return nil
}

func (p *Plugin) Stop() error {
	if err := p.manager.RemoveUpstream(p.apiUpstream.ID); err != nil {
		log.Println("upstream manager emitted an error when attempting to remove the plugin upstream")
	}

	// the api is responsible for decommisioning all upstreams and backends
	// from the parent using the manager. It stops receiving traffic as
	// well once this is called.
	return p.api.Stop()
}

func (p *Plugin) Heartbeat() error {
	log.Println("plugin:mysql-upstreams-api received heartbeat from parent process")
	return nil
}

func (p *Plugin) SetManager(manager upstream_plugin.Manager) error {
	p.rpcManager = manager
	return nil
}

func (p *Plugin) UpstreamMetric(metric *gatekeeper.UpstreamMetric) error {
	log.Println("upstream metric ...")
	return nil
}

func main() {
	plugin := &Plugin{
		config: &Config{},
	}

	if err := upstream_plugin.RunPlugin("mysql-api-upstreams", plugin); err != nil {
		log.Println(err)
	}
}

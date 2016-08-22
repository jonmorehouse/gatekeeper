package main

import (
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	consul "github.com/hashicorp/consul/api"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/gatekeeper/utils"
	upstream_plugin "github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

func NewConsulUpstreams() upstream_plugin.Plugin {
	return &consulUpstreams{
		consulWorkers: 10,

		upstreamIndices: make(map[gatekeeper.UpstreamID]uint64),
		backendIndices:  make(map[gatekeeper.BackendID]uint64),

		stopCh: make(chan struct{}, 1),
	}
}

type consulUpstreams struct {
	manager utils.ServiceContainer

	// this is the last index from consul that we have updated against
	consulConfig  *consul.Config
	consulClient  *consul.Client
	consulIndex   uint64
	consulWorkers int

	upstreamIndices map[gatekeeper.UpstreamID]uint64
	backendIndices  map[gatekeeper.BackendID]uint64

	stopCh chan struct{}
	sync.RWMutex
}

func (c *consulUpstreams) Start() error {
	log.Println("consul-upstreams plugin started ...")

	client, err := consul.NewClient(c.consulConfig)
	log.Println(client, err)
	if err != nil {
		return err
	}

	c.Lock()
	c.consulClient = client
	defer c.Unlock()

	go c.worker()
	return nil
}

func (c *consulUpstreams) Stop() error {
	log.Println("consul-upstreams plugin stopped ...")
	c.stopCh <- struct{}{}
	<-c.stopCh

	// remove all upstreams and all backends from the parent process
	c.RLock()
	defer c.RUnlock()
	for _, upstream := range c.manager.FetchAllUpstreams() {
		if err := c.manager.RemoveUpstream(upstream.ID); err != nil {
			log.Println(err)
		}
	}

	for _, backend := range c.manager.FetchAllBackends() {
		if err := c.manager.RemoveBackend(backend.ID); err != nil {
			log.Println(err)
		}
	}

	return nil
}

func (c *consulUpstreams) Configure(opts map[string]interface{}) error {
	address, ok := opts["upstream-consul-address"]
	if !ok {
		return errors.New("missing -upstream-consul-address flag")
	}
	c.consulConfig = consul.DefaultConfig()
	c.consulConfig.Address = address.(string)

	if scheme, ok := opts["upstream-consul-scheme"]; ok {
		c.consulConfig.Scheme = scheme.(string)
	}

	if datacenter, ok := opts["upstream-consul-datacenter"]; ok {
		c.consulConfig.Datacenter = datacenter.(string)
	}

	if rawWaitTime, ok := opts["upstream-consul-wait-time"]; ok {
		waitTime, err := time.ParseDuration(rawWaitTime.(string))
		if err != nil {
			return err
		}
		c.consulConfig.WaitTime = waitTime
	}

	if token, ok := opts["upstream-consul-token"]; ok {
		c.consulConfig.Token = token.(string)
	}

	return nil
}

func (c *consulUpstreams) SetManager(manager upstream_plugin.Manager) error {
	c.Lock()
	defer c.Unlock()
	c.manager = utils.NewSyncedServiceContainer(manager)
	return nil
}

func (c *consulUpstreams) Heartbeat() error {
	log.Println("consul-upstreams healthcheck ...")
	return nil
}

func (c *consulUpstreams) UpstreamMetric(*gatekeeper.UpstreamMetric) error {
	log.Println("consul-upstreams plugin relies upon consul for healthchecks; noop")
	return nil
}

// fetch all services from consul-agent by using the /v1/catalog/services api
// endpoint. The first time this is called with an uninitialized (zero)
// consulIndex, all services will be fetched. Subsequent calls will block and
// fetch new services since the last written index. Each service corresponds to
// a second HTTP call to /v1/catalog/service/<foo> which returns each backend
// instance of the upstream. This method handles concurrency of requests to the
// upstream by ensuring that we respect the consul-concurrency flag for
// managing how many goroutines to run at once.
func (c *consulUpstreams) fetchServices() error {
	catalog := c.consulClient.Catalog()

	// fetch all services by string name from the configured data center
	services, queryMeta, err := catalog.Services(&consul.QueryOptions{
		Datacenter: c.consulConfig.Datacenter,
		WaitIndex:  c.consulIndex,
		WaitTime:   c.consulConfig.WaitTime,
		Near:       "_agent",
	})

	if err != nil {
		return err
	}

	c.Lock()
	c.consulIndex = queryMeta.LastIndex
	c.Unlock()

	// build a list of jobs to queue up. Each job represents a service/tag
	// representation that we need to queue up.
	jobs := make([][2]string, 0, 0)
	for serviceName, tags := range services {
		for _, tag := range tags {
			jobs = append(jobs, [2]string{serviceName, tag})
		}
	}

	// queue all jobs into a channel to be distributed amongst a series of
	// goroutines to be processed concurrently.
	jobCh := make(chan [2]string, len(jobs))
	for _, job := range jobs {
		jobCh <- job
	}

	var wg sync.WaitGroup
	wg.Add(len(jobs))

	for i := 0; i < c.consulWorkers; i++ {
		go func() {
			for job := range jobCh {
				if err := c.fetchService(job[0], job[1]); err != nil {
					log.Println(err)
				}
				defer wg.Done()
			}
		}()
	}

	wg.Wait()
	close(jobCh)
	return nil
}

// fetches an individual service and all backends for it from the consul agent.
// Each service name corresponds to a unique upstream, and each instance of it
// corresponds to a backend. This method is responsible for writing upstreams
// locally and updating the corresponding index.
func (c *consulUpstreams) fetchService(serviceName, tag string) error {
	catalog := c.consulClient.Catalog()
	instances, _, err := catalog.Service(serviceName, tag, &consul.QueryOptions{
		Datacenter: c.consulConfig.Datacenter,
		WaitIndex:  c.consulIndex,
		WaitTime:   c.consulConfig.WaitTime,
		Near:       "_agent",
	})

	if err != nil {
		return err
	}

	upstream := &gatekeeper.Upstream{
		ID:   gatekeeper.UpstreamID(serviceName),
		Name: serviceName,
		Protocols: []gatekeeper.Protocol{
			gatekeeper.HTTPPublic,
			gatekeeper.HTTPInternal,
			gatekeeper.HTTPSPublic,
			gatekeeper.HTTPSInternal,
		},
		Hostnames: []string{serviceName},
		Prefixes:  []string{serviceName},
	}

	backends := make([]*gatekeeper.Backend, len(instances))

	for idx, instance := range instances {
		address := instance.ServiceAddress
		if instance.ServicePort > 0 {
			address = fmt.Sprintf("%s:%d")
		}
		if address == "" {
			address = instance.Address
		}

		backends[idx] = &gatekeeper.Backend{
			ID:      gatekeeper.BackendID(fmt.Sprintf("%s:%s", upstream.ID, address)),
			Address: address,
			Extra: map[string]interface{}{
				"upstreamID": upstream.ID,
			},
		}
	}

	c.Lock()
	defer c.Unlock()

	// update internal state
	c.upstreamIndices[upstream.ID] = c.consulIndex
	for _, backend := range backends {
		c.backendIndices[backend.ID] = c.consulIndex
	}

	return nil
}

// sync writes all upstreams and backends to the upstream manager. Due to the
// nature with which consul registers and deregisters services, if an upstream
// or backend's index is no longer valid or it has been deemed unhealthy, we
// remove it from the parent.
func (c *consulUpstreams) sync() error {
	// cleanup upstreams from the internal state if they were removed
	// throughout the sync process.
	removedUpstreams := make([]gatekeeper.UpstreamID, 0, 0)
	removedBackends := make([]gatekeeper.BackendID, 0, 0)

	// defer this function so its called last, once the readlock has been surrendered
	defer func() {
		c.Lock()
		defer c.Unlock()

		for _, upstreamID := range removedUpstreams {
			delete(c.upstreamIndices, upstreamID)
			if err := c.manager.RemoveUpstream(upstreamID); err != nil {
				log.Println(err)
			}
		}

		for _, backendID := range removedBackends {
			delete(c.backendIndices, backendID)
			if err := c.manager.RemoveBackend(backendID); err != nil {
				log.Println(err)
			}
		}
	}()

	c.RLock()
	defer c.RUnlock()

	for _, upstream := range c.manager.FetchAllUpstreams() {
		index, ok := c.upstreamIndices[upstream.ID]
		if !ok || index < c.consulIndex {
			removedUpstreams = append(removedUpstreams, upstream.ID)
			continue
		}

		if err := c.manager.AddUpstream(upstream); err != nil {
			log.Println(err)
		}
	}

	for _, backend := range c.manager.FetchAllBackends() {
		index, ok := c.backendIndices[backend.ID]
		// remove the backend if its no longer available or in a weird state
		if !ok || index < c.consulIndex {
			removedBackends = append(removedBackends, backend.ID)
			continue
		}

		upstreamID := backend.Extra["upstreamID"].(gatekeeper.UpstreamID)
		if err := c.manager.AddBackend(upstreamID, backend); err != nil {
			log.Println(err)
		}
	}

	return nil
}

func (c *consulUpstreams) worker() {
	for {
		select {
		case <-c.stopCh:
			break
		default:
			log.Println("syncing...")
			if err := c.fetchServices(); err != nil {
				log.Println(err)
				time.Sleep(time.Second)
				continue
			}
			if err := c.sync(); err != nil {
				log.Println(err)
				time.Sleep(time.Second)
				continue
			}
		}
	}
	close(c.stopCh)
}

func main() {
	plugin := NewConsulUpstreams()
	if err := upstream_plugin.RunPlugin("consul-upstreams", plugin); err != nil {
		log.Fatal(err)
	}
}

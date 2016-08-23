package main

import (
	"encoding/json"
	"errors"
	"net"
	"strconv"
	"strings"
	"time"

	"github.com/docker/engine-api/client"
	"github.com/docker/engine-api/types"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/gatekeeper/utils"
	"golang.org/x/net/context"
)

type services map[*gatekeeper.Upstream][]*gatekeeper.Backend

func joinStrMaps(a, b map[string]string) map[string]string {
	o := make(map[string]string, len(a))

	for _, m := range [2]map[string]string{a, b} {
		for k, v := range m {
			o[k] = v
		}
	}

	return o
}

// getListener returns the first public port found
func getPublicTCPPort(portCfgs []types.Port) (int, error) {
	// look for a TCP port that has a public mapping
	for _, cfg := range portCfgs {
		if cfg.Type == "tcp" && cfg.PublicPort != 0 {
			return cfg.PublicPort, nil
		}
	}

	return 0, errors.New("no listeners found")
}

type Sync interface {
	Sync() error
}

func newDockerSync(client *client.Client, serviceContainer utils.ServiceContainer, defaultTimeout time.Duration, defaultProtocols []gatekeeper.Protocol) Sync {
	return &dockerServices{
		defaultTimeout:   defaultTimeout,
		defaultProtocols: defaultProtocols,
		client:           client,
		serviceContainer: serviceContainer,
	}
}

type dockerServices struct {
	client           *client.Client
	serviceContainer utils.ServiceContainer

	defaultTimeout   time.Duration
	defaultProtocols []gatekeeper.Protocol
}

// sync fetches state from the remote docker engine api, maps the state to the
// correct upstreams and backends and finally synchronize state with a
// ServiceContainer
func (d *dockerServices) Sync() error {
	ctrs, images, err := d.fetchState()
	if err != nil {
		return err
	}

	svcs, err := d.resolveServices(ctrs, images)
	if err != nil {
		return err
	}

	return d.syncServices(svcs, d.serviceContainer)
}

// fetch all state from the docker API that is needed to update the current state
func (d *dockerServices) fetchState() ([]types.Container, []types.Image, error) {
	containers, err := d.client.ContainerList(context.Background(), types.ContainerListOptions{All: true})
	if err != nil {
		return []types.Container(nil), []types.Image(nil), err
	}

	images, err := d.client.ImageList(context.Background(), types.ImageListOptions{All: true})
	if err != nil {
		return []types.Container(nil), []types.Image(nil), err
	}

	return containers, images, nil
}

// accept a list of containers and images, and resolve them into the correct upstreams and backends
func (d *dockerServices) resolveServices(containers []types.Container, images []types.Image) (services, error) {
	svcs := make(services)

	for _, container := range containers {
		if container.Status != "running" {
			continue
		}

		// fetch the port from the container
		if _, err := getPublicTCPPort(container.Ports); err != nil {
			continue
		}

		// fetch the image for this container
		var image types.Image
		for _, img := range images {
			if img.ID == container.ImageID {
				image = img
				break
			}
		}

		// check to ensure the container or image hasn't been flagged to be ignored
		if _, ok := container.Labels["gatekeeper:ignore"]; ok {
			continue
		}
		if _, ok := image.Labels["gatekeeper:ignore"]; ok {
			continue
		}

		labels := joinStrMaps(container.Labels, image.Labels)
		upstream, backend, err := d.resolveService(container, image, labels)
		if err != nil {
			return nil, err
		}

		svcs[upstream] = append(svcs[upstream], backend)
	}

	return svcs, nil
}

// resolveService builds out a backend / upstream from the given container and image
func (d *dockerServices) resolveService(container types.Container, image types.Image, labels map[string]string) (*gatekeeper.Upstream, *gatekeeper.Backend, error) {
	upstream := &gatekeeper.Upstream{}
	backend := &gatekeeper.Backend{}

	// resolve the upstream name from either a label or the container image name
	name, ok := labels["gatekeeper:name"]
	if !ok {
		name, ok = labels["gatekeeper:upstream_name"]
		if !ok {
			name = container.Image
		}
	}
	upstream.Name = name

	// resolve an ID for the upstream from either the upstream_id label or the container Image
	upstreamID, ok := labels["gatekeeper:upstream_id"]
	if !ok {
		upstreamID = container.Image
	}
	upstreamID = upstreamID

	prefixes, ok := labels["gatekeeper:prefixes"]
	if ok {
		upstream.Prefixes = strings.Split(prefixes, ",")
	}

	hostnames, ok := labels["gatekeeper:hostnames"]
	if ok {
		upstream.Hostnames = strings.Split(hostnames, ",")
	}

	// parse protocols from labels or use the default protocols
	protocols, ok := labels["gatekeeper:protocols"]
	if ok {
		prots, err := gatekeeper.ParseProtocols(strings.Split(protocols, ","))
		if err != nil {
			return nil, nil, err
		}
		upstream.Protocols = prots
	} else {
		upstream.Protocols = d.defaultProtocols
	}

	// parse the timeout
	timeout, ok := labels["gatekeeper:timeout"]
	if ok {
		dur, err := time.ParseDuration(timeout)
		if err != nil {
			return nil, nil, err
		}
		upstream.Timeout = dur
	} else {
		upstream.Timeout = d.defaultTimeout
	}

	// parse extra config as json into the upstream.Extra field
	extra, ok := labels["gatekeeper:extra"]
	if ok {
		if err := json.Unmarshal([]byte(extra), &(upstream.Extra)); err != nil {
			return nil, nil, err
		}
	}

	// resolve the backendID from either a label or the container ID
	backendID, ok := labels["gatekeeper:backend_id"]
	if !ok {
		backendID = container.ID
	}
	backend.ID = gatekeeper.BackendID(backendID)

	// fetch a publicTCPPort from the container definition
	port, err := getPublicTCPPort(container.Ports)
	if err != nil {
		return nil, nil, err
	}

	// resolve the broadcastAddress
	broadcastAddress, ok := labels["gatekeeper:broadcast_address"]
	if !ok {
		broadcastHost, ok := labels["gatekeeper:broadcast_host"]
		if !ok {
			broadcastAddress = "localhost"
		}
		broadcastAddress = net.JoinHostPort(broadcastHost, strconv.Itoa(port))
	}
	backend.Address = broadcastAddress

	return upstream, backend, nil
}

// accept a map of upstreams -> backends and synchronize them with a ServiceContainer
func (d *dockerServices) syncServices(svcs services, ctr utils.ServiceContainer) error {
	upstreams := make(map[gatekeeper.UpstreamID]*gatekeeper.Upstream)
	backends := make(map[gatekeeper.BackendID]*gatekeeper.Backend)

	// loop through all current upstreams/backends
	for _, upstream := range ctr.FetchAllUpstreams() {
		upstreams[upstream.ID] = upstream
		bcknds, err := ctr.FetchBackends(upstream.ID)
		if err != nil {
			continue
		}

		for _, backend := range bcknds {
			backends[backend.ID] = backend
		}
	}

	for upstream, bcknds := range svcs {
		ctr.AddUpstream(upstream)
		delete(upstreams, upstream.ID)

		for _, backend := range bcknds {
			ctr.AddBackend(upstream.ID, backend)
			delete(backends, backend.ID)
		}
	}

	return nil
}

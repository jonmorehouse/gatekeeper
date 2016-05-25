package gatekeeper

import (
	"fmt"

	"github.com/jonmorehouse/gatekeeper/plugin/upstream"
)

type UpstreamPlugin interface {
	// start is used by the upstreamDirector to start a _connected_
	// plugin's lifecycle. In most cases, this involves making an RPC call
	// to the plugin and starting the "event flow" of upstream/backend info
	// from the plugin to this struct.
	Start() error
	Stop() error

	// TODO: add a Restart method to this interface
}

// UpstreamPublisher starts, maintains and wraps an UpstreamPlugin, accepting
// events from the plugin. Each plugin event is serialized into the correct
// type and published to the broadcaster type to ensure that all listeners
// receive the message correctly.
type UpstreamPluginPublisher struct {
	opts        PluginOpts
	plugin      upstream.Plugin
	broadcaster EventBroadcaster

	// keep a tally of the upstreams / plugins we've seen here
	knownUpstreams map[upstream.UpstreamID]interface{}
	knownBackends  map[upstream.BackendID]interface{}
}

func NewUpstreamPluginPublisher(opts PluginOpts, broadcaster EventBroadcaster) (UpstreamPlugin, error) {
	plugin, err := upstream.NewClient(opts.Name, opts.Cmd)
	if err != nil {
		return nil, err
	}
	if err := plugin.Configure(opts.Opts); err != nil {
		return nil, err
	}

	return &UpstreamPluginPublisher{
		broadcaster: broadcaster,
		opts:        opts,
		plugin:      plugin,
	}, nil
}

func (p *UpstreamPluginPublisher) Start() error {
	if err := p.plugin.Start(p); err != nil {
		return err
	}
	return nil
}

func (p *UpstreamPluginPublisher) Stop() error {
	if err := p.plugin.Stop(); err != nil {
		return err
	}
	return nil
}

func (p *UpstreamPluginPublisher) AddUpstream(pluginUpstream upstream.Upstream) (upstream.UpstreamID, error) {
	u := PluginUpstreamToUpstream(pluginUpstream, upstream.NilUpstreamID)
	p.knownUpstreams[u.ID] = struct{}{}

	return u.ID, p.broadcaster.Publish(UpstreamEvent{
		EventType:  UpstreamAdded,
		Upstream:   u,
		UpstreamID: u.ID,
	})
}

func (p *UpstreamPluginPublisher) RemoveUpstream(uID upstream.UpstreamID) error {
	if _, ok := p.knownUpstreams[uID]; !ok {
		return fmt.Errorf("Unknown upstream")
	}

	delete(p.knownUpstreams, uID)
	return p.broadcaster.Publish(UpstreamEvent{
		EventType:  UpstreamRemoved,
		UpstreamID: uID,
	})
}

func (p *UpstreamPluginPublisher) AddBackend(uID upstream.UpstreamID, pluginBackend upstream.Backend) (upstream.BackendID, error) {
	if _, ok := p.knownUpstreams[uID]; !ok {
		return upstream.NilBackendID, fmt.Errorf("Unknown upstream")
	}

	backend := PluginBackendToBackend(pluginBackend, upstream.NilBackendID)
	p.knownBackends[backend.ID] = struct{}{}
	return backend.ID, p.broadcaster.Publish(UpstreamEvent{
		EventType:  BackendAdded,
		UpstreamID: uID,
		BackendID:  backend.ID,
		Backend:    backend,
	})
}

func (p *UpstreamPluginPublisher) RemoveBackend(bID upstream.BackendID) error {
	if _, ok := p.knownBackends[bID]; !ok {
		return fmt.Errorf("Unknown backend")
	}
	return p.broadcaster.Publish(UpstreamEvent{
		EventType: BackendRemoved,
		BackendID: bID,
	})
}

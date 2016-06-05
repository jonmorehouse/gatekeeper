package gatekeeper

import "fmt"

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
	if err := p.pluginManager.Start(); err != nil {
		return err
	}

	// for each plugin, we call the SetupServer RPC connection
	plugins, err := p.pluginManager.All()
	if err != nil {
		return err
	}

	for _, plugin := range plugins() {

	}

	return nil
}

func (p *UpstreamPluginPublisher) Stop() error {
	if err := p.pluginManager.Stop(); err != nil {
		return err
	}
	return nil
}

func (p *UpstreamPluginPublisher) AddUpstream(pluginUpstream upstream.Upstream) (upstream.UpstreamID, error) {
	u := PluginUpstreamToUpstream(pluginUpstream, upstream.NilUpstreamID)
	p.knownUpstreams[upstream.UpstreamID(u.ID)] = struct{}{}

	return upstream.UpstreamID(u.ID), p.broadcaster.Publish(UpstreamEvent{
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
		UpstreamID: UpstreamID(uID),
	})
}

func (p *UpstreamPluginPublisher) AddBackend(uID upstream.UpstreamID, pluginBackend upstream.Backend) (upstream.BackendID, error) {
	if _, ok := p.knownUpstreams[uID]; !ok {
		return upstream.NilBackendID, fmt.Errorf("Unknown upstream")
	}

	backend := PluginBackendToBackend(pluginBackend, upstream.NilBackendID)
	p.knownBackends[upstream.BackendID(backend.ID)] = struct{}{}
	return upstream.BackendID(backend.ID), p.broadcaster.Publish(UpstreamEvent{
		EventType:  BackendAdded,
		UpstreamID: UpstreamID(uID),
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
		BackendID: BackendID(bID),
	})
}

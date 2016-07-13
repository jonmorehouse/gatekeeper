package metric

import (
	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"

	"github.com/jonmorehouse/gatekeeper/internal"
)

// Plugin is the interface which a plugin will implement and pass to `RunPlugin`
type Plugin interface {
	// internal.Plugin exposes the following methods, per:
	// https://github.com/jonmorehouse/gateekeeper/tree/master/internal/plugin.go
	//
	// Start() error
	// Stop() error
	// Heartbeat() error
	// Configure(map[string]interface{}) error
	//
	internal.BasePlugin

	EventMetric(*gatekeeper.EventMetric) error
	ProfilingMetric(*gatekeeper.ProfilingMetric) error
	PluginMetric(*gatekeeper.PluginMetric) error
	RequestMetric(*gatekeeper.RequestMetric) error
	UpstreamMetric(*gatekeeper.UpstreamMetric) error
}

// PluginClient in this case is the gatekeeper/core application. PluginClient
// is the interface that the user of this plugin sees and is simply a wrapper
// around *RPCClient. This is merely a wrapper which returns a clean interface
// with error interfaces instead of *gatekeeper.Error types
type PluginClient interface {
	internal.BasePluginClient

	// Metrics are batch written over RPC on the client side, so as to allow for less noise over the wire
	WriteEventMetrics([]*gatekeeper.EventMetric) []error
	WriteProfilingMetrics([]*gatekeeper.ProfilingMetric) []error
	WritePluginMetrics([]*gatekeeper.PluginMetric) []error
	WriteRequestMetrics([]*gatekeeper.RequestMetric) []error
	WriteUpstreamMetrics([]*gatekeeper.UpstreamMetric) []error
}

func NewPluginClient(rpcClient *RPCClient, client *plugin.Client) PluginClient {
	return &pluginClient{
		rpcClient,
		internal.NewBasePluginClient(rpcClient, client),
	}
}

type pluginClient struct {
	pluginRPC *RPCClient
	internal.BasePluginClient
}

func (p *pluginClient) WriteEventMetrics(metrics []*gatekeeper.EventMetric) []error {
	if errs := p.pluginRPC.EventMetric(metrics); errs != nil {
		return gatekeeper.ErrorsToErrors(errs)
	}
	return nil
}

func (p *pluginClient) WriteProfilingMetrics(metrics []*gatekeeper.ProfilingMetric) []error {
	if errs := p.pluginRPC.ProfilingMetric(metrics); errs != nil {
		return gatekeeper.ErrorsToErrors(errs)
	}
	return nil
}

func (p *pluginClient) WritePluginMetrics(metrics []*gatekeeper.PluginMetric) []error {
	if errs := p.pluginRPC.PluginMetric(metrics); errs != nil {
		return gatekeeper.ErrorsToErrors(errs)
	}
	return nil
}

func (p *pluginClient) WriteRequestMetrics(metrics []*gatekeeper.RequestMetric) []error {
	if errs := p.pluginRPC.RequestMetric(metrics); errs != nil {
		return gatekeeper.ErrorsToErrors(errs)
	}
	return nil
}

func (p *pluginClient) WriteUpstreamMetrics(metrics []*gatekeeper.UpstreamMetric) []error {
	if errs := p.pluginRPC.UpstreamMetric(metrics); errs != nil {
		return gatekeeper.ErrorsToErrors(errs)
	}
	return nil
}

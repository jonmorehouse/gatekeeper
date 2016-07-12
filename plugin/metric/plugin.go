package metric

import (
	"github.com/jonmorehouse/gatekeeper/shared"

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

	EventMetric(*shared.EventMetric) error
	ProfilingMetric(*shared.ProfilingMetric) error
	PluginMetric(*shared.PluginMetric) error
	RequestMetric(*shared.RequestMetric) error
	UpstreamMetric(*shared.UpstreamMetric) error
}

// PluginClient in this case is the gatekeeper/core application. PluginClient
// is the interface that the user of this plugin sees and is simply a wrapper
// around *RPCClient. This is merely a wrapper which returns a clean interface
// with error interfaces instead of *shared.Error types
type PluginClient interface {
	internal.BasePlugin

	// Metrics are batch written over RPC on the client side, so as to allow for less noise over the wire
	WriteEventMetrics([]*shared.EventMetric) []error
	WriteProfilingMetrics([]*shared.ProfilingMetric) []error
	WritePluginMetrics([]*shared.PluginMetric) []error
	WriteRequestMetrics([]*shared.RequestMetric) []error
	WriteUpstreamMetrics([]*shared.UpstreamMetric) []error
}

func NewPluginClient(rpcClient *RPCClient) PluginClient {
	return &pluginClient{
		rpcClient,
		internal.NewBasePluginClient(rpcClient),
	}
}

type pluginClient struct {
	pluginRPC *RPCClient
	*internal.BasePluginClient
}

func (p *pluginClient) WriteEventMetrics(metrics []*shared.EventMetric) []error {
	errs := p.pluginRPC.EventMetric(metrics)
	return shared.ErrorsToErrors(errs)
}

func (p *pluginClient) WriteProfilingMetrics(metrics []*shared.ProfilingMetric) []error {
	errs := p.pluginRPC.ProfilingMetric(metrics)
	return shared.ErrorsToErrors(errs)
}

func (p *pluginClient) WritePluginMetrics(metrics []*shared.PluginMetric) []error {
	errs := p.pluginRPC.PluginMetric(metrics)
	return shared.ErrorsToErrors(errs)
}

func (p *pluginClient) WriteRequestMetrics(metrics []*shared.RequestMetric) []error {
	errs := p.pluginRPC.RequestMetric(metrics)
	return shared.ErrorsToErrors(errs)
}

func (p *pluginClient) WriteUpstreamMetrics(metrics []*shared.UpstreamMetric) []error {
	errs := p.pluginRPC.UpstreamMetric(metrics)
	return shared.ErrorsToErrors(errs)
}

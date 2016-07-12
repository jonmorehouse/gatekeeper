package metric

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/internal"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type eventMetricArgs struct {
	Metrics []*gatekeeper.EventMetric
}
type eventMetricResp struct {
	Errs []*gatekeeper.Error
}

type profilingMetricArgs struct {
	Metrics []*gatekeeper.ProfilingMetric
}
type profilingMetricResp struct {
	Errs []*gatekeeper.Error
}

type pluginMetricArgs struct {
	Metrics []*gatekeeper.PluginMetric
}
type pluginMetricResp struct {
	Errs []*gatekeeper.Error
}

type requestMetricArgs struct {
	Metrics []*gatekeeper.RequestMetric
}
type requestMetricResp struct {
	Errs []*gatekeeper.Error
}

type upstreamMetricArgs struct {
	Metrics []*gatekeeper.UpstreamMetric
}
type upstreamMetricResp struct {
	Errs []*gatekeeper.Error
}

// PluginRPC is a representation of the Plugin interface that is RPC safe. It
// embeds an internal.BasePluginRPC which handles the basic RPC client
// communications of the `Start`, `Stop`, `Configure` and `Heartbeat` methods.
type RPCClient struct {
	broker *plugin.MuxBroker
	client *rpc.Client

	*internal.BasePluginRPCClient
}

func (c *RPCClient) ProfilingMetric(metrics []*gatekeeper.ProfilingMetric) []*gatekeeper.Error {
	callArgs := profilingMetricArgs{
		Metrics: metrics,
	}
	callResp := profilingMetricResp{}

	if err := c.client.Call("Plugin.ProfilingMetric", &callArgs, &callResp); err != nil {
		return []*gatekeeper.Error{gatekeeper.NewError(err)}
	}

	return callResp.Errs
}

func (c *RPCClient) PluginMetric(metrics []*gatekeeper.PluginMetric) []*gatekeeper.Error {
	callArgs := pluginMetricArgs{
		Metrics: metrics,
	}
	callResp := pluginMetricResp{}

	if err := c.client.Call("Plugin.PluginMetric", &callArgs, &callResp); err != nil {
		return []*gatekeeper.Error{gatekeeper.NewError(err)}
	}

	return callResp.Errs
}

func (c *RPCClient) RequestMetric(metrics []*gatekeeper.RequestMetric) []*gatekeeper.Error {
	callArgs := requestMetricArgs{
		Metrics: metrics,
	}
	callResp := requestMetricResp{}

	if err := c.client.Call("Plugin.RequestMetric", &callArgs, &callResp); err != nil {
		return []*gatekeeper.Error{gatekeeper.NewError(err)}
	}

	return callResp.Errs
}

func (c *RPCClient) UpstreamMetric(metrics []*gatekeeper.UpstreamMetric) []*gatekeeper.Error {
	callArgs := upstreamMetricArgs{
		Metrics: metrics,
	}
	callResp := upstreamMetricResp{}

	if err := c.client.Call("Plugin.UpstreamMetric", &callArgs, &callResp); err != nil {
		return []*gatekeeper.Error{gatekeeper.NewError(err)}
	}

	return callResp.Errs
}

type RPCServer struct {
	impl   Plugin
	broker *plugin.MuxBroker

	*internal.BasePluginRPCServer
}

func (s *RPCServer) EventMetric(args *eventMetricArgs, resp *eventMetricResp) error {
	errs := make([]*gatekeeper.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.EventMetric(metric); err != nil {
			errs = append(errs, gatekeeper.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (s *RPCServer) ProfilingMetric(args *profilingMetricArgs, resp *profilingMetricResp) error {
	errs := make([]*gatekeeper.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.ProfilingMetric(metric); err != nil {
			errs = append(errs, gatekeeper.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (s *RPCServer) PluginMetric(args *pluginMetricArgs, resp *pluginMetricResp) error {
	errs := make([]*gatekeeper.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.PluginMetric(metric); err != nil {
			errs = append(errs, gatekeeper.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (s *RPCServer) RequestMetric(args *requestMetricArgs, resp *requestMetricResp) error {
	errs := make([]*gatekeeper.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.RequestMetric(metric); err != nil {
			errs = append(errs, gatekeeper.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (s *RPCServer) UpstreamMetric(args *upstreamMetricArgs, resp *upstreamMetricResp) error {
	errs := make([]*gatekeeper.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.UpstreamMetric(metric); err != nil {
			errs = append(errs, gatekeeper.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (c *RPCClient) EventMetric(metrics []*gatekeeper.EventMetric) []*gatekeeper.Error {
	callArgs := eventMetricArgs{
		Metrics: metrics,
	}
	callResp := eventMetricResp{}

	if err := c.client.Call("Plugin.EventMetric", &callArgs, &callResp); err != nil {
		return []*gatekeeper.Error{gatekeeper.NewError(err)}
	}

	return callResp.Errs
}

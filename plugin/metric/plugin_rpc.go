package metric

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/internal"
)

type EventMetricArgs struct {
	Metrics []*gatekeeper.EventMetric
}
type EventMetricResp struct {
	Errs []*gatekeeper.Error
}

type ProfilingMetricArgs struct {
	Metrics []*gatekeeper.ProfilingMetric
}
type ProfilingMetricResp struct {
	Errs []*gatekeeper.Error
}

type PluginMetricArgs struct {
	Metrics []*gatekeeper.PluginMetric
}
type PluginMetricResp struct {
	Errs []*gatekeeper.Error
}

type RequestMetricArgs struct {
	Metrics []*gatekeeper.RequestMetric
}
type RequestMetricResp struct {
	Errs []*gatekeeper.Error
}

type UpstreamMetricArgs struct {
	Metrics []*gatekeeper.UpstreamMetric
}
type UpstreamMetricResp struct {
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
	callArgs := ProfilingMetricArgs{
		Metrics: metrics,
	}
	callResp := ProfilingMetricResp{}

	if err := c.client.Call("Plugin.ProfilingMetric", &callArgs, &callResp); err != nil {
		return []*gatekeeper.Error{gatekeeper.NewError(err)}
	}

	return callResp.Errs
}

func (c *RPCClient) PluginMetric(metrics []*gatekeeper.PluginMetric) []*gatekeeper.Error {
	callArgs := PluginMetricArgs{
		Metrics: metrics,
	}
	callResp := PluginMetricResp{}

	if err := c.client.Call("Plugin.PluginMetric", &callArgs, &callResp); err != nil {
		return []*gatekeeper.Error{gatekeeper.NewError(err)}
	}

	return callResp.Errs
}

func (c *RPCClient) RequestMetric(metrics []*gatekeeper.RequestMetric) []*gatekeeper.Error {
	callArgs := RequestMetricArgs{
		Metrics: metrics,
	}
	callResp := RequestMetricResp{}

	if err := c.client.Call("Plugin.RequestMetric", &callArgs, &callResp); err != nil {
		return []*gatekeeper.Error{gatekeeper.NewError(err)}
	}

	return callResp.Errs
}

func (c *RPCClient) UpstreamMetric(metrics []*gatekeeper.UpstreamMetric) []*gatekeeper.Error {
	callArgs := UpstreamMetricArgs{
		Metrics: metrics,
	}
	callResp := UpstreamMetricResp{}

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

func (s *RPCServer) EventMetric(args *EventMetricArgs, resp *EventMetricResp) error {
	errs := make([]*gatekeeper.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.EventMetric(metric); err != nil {
			errs = append(errs, gatekeeper.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (s *RPCServer) ProfilingMetric(args *ProfilingMetricArgs, resp *ProfilingMetricResp) error {
	errs := make([]*gatekeeper.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.ProfilingMetric(metric); err != nil {
			errs = append(errs, gatekeeper.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (s *RPCServer) PluginMetric(args *PluginMetricArgs, resp *PluginMetricResp) error {
	errs := make([]*gatekeeper.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.PluginMetric(metric); err != nil {
			errs = append(errs, gatekeeper.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (s *RPCServer) RequestMetric(args *RequestMetricArgs, resp *RequestMetricResp) error {
	errs := make([]*gatekeeper.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.RequestMetric(metric); err != nil {
			errs = append(errs, gatekeeper.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (s *RPCServer) UpstreamMetric(args *UpstreamMetricArgs, resp *UpstreamMetricResp) error {
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
	callArgs := EventMetricArgs{
		Metrics: metrics,
	}
	callResp := EventMetricResp{}

	if err := c.client.Call("Plugin.EventMetric", &callArgs, &callResp); err != nil {
		return []*gatekeeper.Error{gatekeeper.NewError(err)}
	}

	return callResp.Errs
}

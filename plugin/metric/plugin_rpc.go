package metric

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type StartArgs struct{}
type StartResp struct {
	Err *shared.Error
}

type StopArgs struct{}
type StopResp struct {
	Err *shared.Error
}

type ConfigureArgs struct {
	Opts map[string]interface{}
}
type ConfigureResp struct {
	Err *shared.Error
}

type HeartbeatArgs struct{}
type HeartbeatResp struct {
	Err *shared.Error
}

type EventMetricArgs struct {
	Metrics []*shared.EventMetric
}
type EventMetricResp struct {
	Errs []*shared.Error
}

type ProfilingMetricArgs struct {
	Metrics []*shared.ProfilingMetric
}
type ProfilingMetricResp struct {
	Errs []*shared.Error
}

type PluginMetricArgs struct {
	Metrics []*shared.PluginMetric
}
type PluginMetricResp struct {
	Errs []*shared.Error
}

type RequestMetricArgs struct {
	Metrics []*shared.RequestMetric
}
type RequestMetricResp struct {
	Errs []*shared.Error
}

type UpstreamMetricArgs struct {
	Metrics []*shared.UpstreamMetric
}
type UpstreamMetricResp struct {
	Errs []*shared.Error
}

// implement the RPC server which the plugin runs, mapping to the Plugin
// interface specified locally
type RPCServer struct {
	impl   Plugin
	broker *plugin.MuxBroker
}

func (s *RPCServer) Start(args *StartArgs, resp *StartResp) error {
	err := s.impl.Start()
	resp.Err = shared.NewError(err)
	return nil
}

func (s *RPCServer) Stop(args *StopArgs, resp *StopResp) error {
	err := s.impl.Stop()
	resp.Err = shared.NewError(err)
	return nil
}

func (s *RPCServer) Heartbeat(args *HeartbeatArgs, resp *HeartbeatResp) error {
	err := s.impl.Heartbeat()
	resp.Err = shared.NewError(err)
	return nil
}

func (s *RPCServer) Configure(args *ConfigureArgs, resp *ConfigureResp) error {
	err := s.impl.Configure(args.Opts)
	resp.Err = shared.NewError(err)
	return nil
}

func (s *RPCServer) EventMetric(args *EventMetricArgs, resp *EventMetricResp) error {
	errs := make([]*shared.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.EventMetric(metric); err != nil {
			errs = append(errs, shared.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (s *RPCServer) ProfilingMetric(args *ProfilingMetricArgs, resp *ProfilingMetricResp) error {
	errs := make([]*shared.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.ProfilingMetric(metric); err != nil {
			errs = append(errs, shared.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (s *RPCServer) PluginMetric(args *PluginMetricArgs, resp *PluginMetricResp) error {
	errs := make([]*shared.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.PluginMetric(metric); err != nil {
			errs = append(errs, shared.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (s *RPCServer) RequestMetric(args *RequestMetricArgs, resp *RequestMetricResp) error {
	errs := make([]*shared.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.RequestMetric(metric); err != nil {
			errs = append(errs, shared.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

func (s *RPCServer) UpstreamMetric(args *UpstreamMetricArgs, resp *UpstreamMetricResp) error {
	errs := make([]*shared.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.UpstreamMetric(metric); err != nil {
			errs = append(errs, shared.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

type RPCClient struct {
	broker *plugin.MuxBroker
	client *rpc.Client
}

func (c *RPCClient) Start() *shared.Error {
	callArgs := StartArgs{}
	callResp := StartResp{}
	if err := c.client.Call("Plugin.Start", &callArgs, &callResp); err != nil {
		return shared.NewError(err)
	}
	return callResp.Err
}

func (c *RPCClient) Stop() *shared.Error {
	callArgs := StopArgs{}
	callResp := StopResp{}
	if err := c.client.Call("Plugin.Stop", &callArgs, &callResp); err != nil {
		return shared.NewError(err)
	}
	return callResp.Err
}

func (c *RPCClient) Heartbeat() *shared.Error {
	callArgs := HeartbeatArgs{}
	callResp := HeartbeatResp{}
	if err := c.client.Call("Plugin.Heartbeat", &callArgs, &callResp); err != nil {
		return shared.NewError(err)
	}
	return callResp.Err
}

func (c *RPCClient) Configure(opts map[string]interface{}) *shared.Error {
	callArgs := ConfigureArgs{
		Opts: opts,
	}
	callResp := ConfigureResp{}
	if err := c.client.Call("Plugin.Configure", &callArgs, &callResp); err != nil {
		return shared.NewError(err)
	}
	return callResp.Err
}

func (c *RPCClient) EventMetric(metrics []*shared.EventMetric) []*shared.Error {
	callArgs := EventMetricArgs{
		Metrics: metrics,
	}
	callResp := EventMetricResp{}

	if err := c.client.Call("Plugin.EventMetric", &callArgs, &callResp); err != nil {
		return []*shared.Error{shared.NewError(err)}
	}

	return callResp.Errs
}

func (c *RPCClient) ProfilingMetric(metrics []*shared.ProfilingMetric) []*shared.Error {
	callArgs := ProfilingMetricArgs{
		Metrics: metrics,
	}
	callResp := ProfilingMetricResp{}

	if err := c.client.Call("Plugin.ProfilingMetric", &callArgs, &callResp); err != nil {
		return []*shared.Error{shared.NewError(err)}
	}

	return callResp.Errs
}

func (c *RPCClient) PluginMetric(metrics []*shared.PluginMetric) []*shared.Error {
	callArgs := PluginMetricArgs{
		Metrics: metrics,
	}
	callResp := PluginMetricResp{}

	if err := c.client.Call("Plugin.PluginMetric", &callArgs, &callResp); err != nil {
		return []*shared.Error{shared.NewError(err)}
	}

	return callResp.Errs
}

func (c *RPCClient) RequestMetric(metrics []*shared.RequestMetric) []*shared.Error {
	callArgs := RequestMetricArgs{
		Metrics: metrics,
	}
	callResp := RequestMetricResp{}

	if err := c.client.Call("Plugin.RequestMetric", &callArgs, &callResp); err != nil {
		return []*shared.Error{shared.NewError(err)}
	}

	return callResp.Errs
}

func (c *RPCClient) UpstreamMetric(metrics []*shared.UpstreamMetric) []*shared.Error {
	callArgs := UpstreamMetricArgs{
		Metrics: metrics,
	}
	callResp := UpstreamMetricResp{}

	if err := c.client.Call("Plugin.UpstreamMetric", &callArgs, &callResp); err != nil {
		return []*shared.Error{shared.NewError(err)}
	}

	return callResp.Errs
}

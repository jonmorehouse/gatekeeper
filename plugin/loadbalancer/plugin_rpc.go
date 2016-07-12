package loadbalancer

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/internal"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type addBackendArgs struct {
	Backend  *gatekeeper.Backend
	Upstream gatekeeper.UpstreamID
}
type addBackendResp struct {
	Err *gatekeeper.Error
}

type removeBackendArgs struct {
	Backend *gatekeeper.Backend
}
type removeBackendResp struct {
	Err *gatekeeper.Error
}

type upstreamMetricArgs struct {
	Metrics []*gatekeeper.UpstreamMetric
}
type upstreamMetricResp struct {
	Errs []*gatekeeper.Error
}

type getBackendArgs struct {
	Upstream gatekeeper.UpstreamID
}
type getBackendResp struct {
	Backend *gatekeeper.Backend
	Err     *gatekeeper.Error
}

// PluginRPC is a representation of the Plugin interface that is RPC safe. It
// embeds an internal.BasePluginRPC which handles the basic RPC client
// communications of the `Start`, `Stop`, `Configure` and `Heartbeat` methods.
type RPCClient struct {
	broker *plugin.MuxBroker
	client *rpc.Client

	*internal.BasePluginRPCClient
}

func (c *RPCClient) AddBackend(upstream gatekeeper.UpstreamID, backend *gatekeeper.Backend) *gatekeeper.Error {
	callArgs := addBackendArgs{
		Upstream: upstream,
		Backend:  backend,
	}
	callResp := addBackendResp{}
	if err := c.client.Call("Plugin.AddBackend", &callArgs, &callResp); err != nil {
		return gatekeeper.NewError(err)
	}
	return callResp.Err
}

func (c *RPCClient) RemoveBackend(backend *gatekeeper.Backend) *gatekeeper.Error {
	callArgs := removeBackendArgs{
		Backend: backend,
	}
	callResp := removeBackendResp{}
	if err := c.client.Call("Plugin.RemoveBackend", &callArgs, &callResp); err != nil {
		return gatekeeper.NewError(err)
	}
	return callResp.Err
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

func (c *RPCClient) GetBackend(upstream gatekeeper.UpstreamID) (*gatekeeper.Backend, *gatekeeper.Error) {
	callArgs := getBackendArgs{
		Upstream: upstream,
	}
	callResp := getBackendResp{}
	if err := c.client.Call("Plugin.GetBackend", &callArgs, &callResp); err != nil {
		return nil, gatekeeper.NewError(err)
	}
	return callResp.Backend, callResp.Err
}

type RPCServer struct {
	impl   Plugin
	broker *plugin.MuxBroker

	*internal.BasePluginRPCServer
}

func (s *RPCServer) AddBackend(args *addBackendArgs, resp *addBackendResp) error {
	resp.Err = gatekeeper.NewError(s.impl.AddBackend(args.Upstream, args.Backend))
	return nil
}

func (s *RPCServer) RemoveBackend(args *removeBackendArgs, resp *removeBackendResp) error {
	if err := s.impl.RemoveBackend(args.Backend); err != nil {
		resp.Err = gatekeeper.NewError(err)
	}
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

func (s *RPCServer) GetBackend(args *getBackendArgs, resp *getBackendResp) error {
	backend, err := s.impl.GetBackend(args.Upstream)
	resp.Backend = backend
	resp.Err = gatekeeper.NewError(err)
	return nil
}

package router

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/internal"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type addUpstreamArgs struct {
	Upstream *shared.Upstream
}
type addUpstreamResp struct {
	Err *shared.Error
}

type removeUpstreamArgs struct {
	UpstreamID shared.UpstreamID
}
type removeUpstreamResp struct {
	Err *shared.Error
}

type routeRequestArgs struct {
	Req *shared.Request
}
type routeRequestResp struct {
	Upstream *shared.Upstream
	Err      *shared.Error
}

// PluginRPC is a representation of the Plugin interface that is RPC safe. It
// embeds an internal.BasePluginRPC which handles the basic RPC client
// communications of the `Start`, `Stop`, `Configure` and `Heartbeat` methods.
type RPCClient struct {
	broker *plugin.MuxBroker
	client *rpc.Client

	*internal.BasePluginRPCClient
}

func (c *RPCClient) AddUpstream(upstream *shared.Upstream) *shared.Error {
	args := &addUpstreamArgs{
		Upstream: upstream,
	}
	resp := &addUpstreamResp{}

	if err := c.client.Call("Plugin.AddUpstream", args, resp); err != nil {
		return shared.NewError(err)
	}

	return resp.Err
}

func (c *RPCClient) RemoveUpstream(upstreamID shared.UpstreamID) *shared.Error {
	args := &removeUpstreamArgs{
		UpstreamID: upstreamID,
	}
	resp := &removeUpstreamResp{}

	if err := c.client.Call("Plugin.RemoveUpstream", args, resp); err != nil {
		return shared.NewError(err)
	}

	return resp.Err
}

func (c *RPCClient) RouteRequest(req *shared.Request) (*shared.Upstream, *shared.Error) {
	args := &routeRequestArgs{
		Req: req,
	}
	resp := &routeRequestResp{}

	if err := c.client.Call("Plugin.RouteRequest", args, resp); err != nil {
		return nil, shared.NewError(err)
	}

	return resp.Upstream, resp.Err

}

type RPCServer struct {
	impl   Plugin
	broker *plugin.MuxBroker

	*internal.BasePluginRPCServer
}

func (s *RPCServer) AddUpstream(args *addUpstreamArgs, resp *addUpstreamResp) error {
	if err := s.impl.AddUpstream(args.Upstream); err != nil {
		resp.Err = shared.NewError(err)
	}
	return nil
}

func (s *RPCServer) RemoveUpstream(args *removeUpstreamArgs, resp *removeUpstreamResp) error {
	if err := s.impl.RemoveUpstream(args.UpstreamID); err != nil {
		resp.Err = shared.NewError(err)
	}
	return nil
}

func (s *RPCServer) RouteRequest(args *routeRequestArgs, resp *routeRequestResp) error {
	upstream, err := s.impl.RouteRequest(args.Req)
	if err != nil {
		resp.Err = shared.NewError(err)
		return nil
	}
	resp.Upstream = upstream
	return nil
}

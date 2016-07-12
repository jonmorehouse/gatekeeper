package router

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/internal"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type addUpstreamArgs struct {
	Upstream *gatekeeper.Upstream
}
type addUpstreamResp struct {
	Err *gatekeeper.Error
}

type removeUpstreamArgs struct {
	UpstreamID gatekeeper.UpstreamID
}
type removeUpstreamResp struct {
	Err *gatekeeper.Error
}

type routeRequestArgs struct {
	Req *gatekeeper.Request
}
type routeRequestResp struct {
	Upstream *gatekeeper.Upstream
	Req      *gatekeeper.Request
	Err      *gatekeeper.Error
}

// PluginRPC is a representation of the Plugin interface that is RPC safe. It
// embeds an internal.BasePluginRPC which handles the basic RPC client
// communications of the `Start`, `Stop`, `Configure` and `Heartbeat` methods.
type RPCClient struct {
	broker *plugin.MuxBroker
	client *rpc.Client

	*internal.BasePluginRPCClient
}

func (c *RPCClient) AddUpstream(upstream *gatekeeper.Upstream) *gatekeeper.Error {
	args := &addUpstreamArgs{
		Upstream: upstream,
	}
	resp := &addUpstreamResp{}

	if err := c.client.Call("Plugin.AddUpstream", args, resp); err != nil {
		return gatekeeper.NewError(err)
	}

	return resp.Err
}

func (c *RPCClient) RemoveUpstream(upstreamID gatekeeper.UpstreamID) *gatekeeper.Error {
	args := &removeUpstreamArgs{
		UpstreamID: upstreamID,
	}
	resp := &removeUpstreamResp{}

	if err := c.client.Call("Plugin.RemoveUpstream", args, resp); err != nil {
		return gatekeeper.NewError(err)
	}

	return resp.Err
}

func (c *RPCClient) RouteRequest(req *gatekeeper.Request) (*gatekeeper.Upstream, *gatekeeper.Request, *gatekeeper.Error) {
	args := &routeRequestArgs{
		Req: req,
	}
	resp := &routeRequestResp{}

	if err := c.client.Call("Plugin.RouteRequest", args, resp); err != nil {
		return nil, args.Req, gatekeeper.NewError(err)
	}

	return resp.Upstream, resp.Req, resp.Err

}

type RPCServer struct {
	impl   Plugin
	broker *plugin.MuxBroker

	*internal.BasePluginRPCServer
}

func (s *RPCServer) AddUpstream(args *addUpstreamArgs, resp *addUpstreamResp) error {
	if err := s.impl.AddUpstream(args.Upstream); err != nil {
		resp.Err = gatekeeper.NewError(err)
	}
	return nil
}

func (s *RPCServer) RemoveUpstream(args *removeUpstreamArgs, resp *removeUpstreamResp) error {
	if err := s.impl.RemoveUpstream(args.UpstreamID); err != nil {
		resp.Err = gatekeeper.NewError(err)
	}
	return nil
}

func (s *RPCServer) RouteRequest(args *routeRequestArgs, resp *routeRequestResp) error {
	upstream, req, err := s.impl.RouteRequest(args.Req)
	if err != nil {
		resp.Err = gatekeeper.NewError(err)
		return nil
	}
	resp.Upstream = upstream
	resp.Req = req
	return nil
}

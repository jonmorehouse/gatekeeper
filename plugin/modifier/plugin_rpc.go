package modifier

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
	"github.com/jonmorehouse/gatekeeper/internal"
)

type ModifyRequestArgs struct {
	Request *gatekeeper.Request
}
type ModifyRequestResp struct {
	Request *gatekeeper.Request
	Err     *gatekeeper.Error
}

type ModifyResponseArgs struct {
	Request  *gatekeeper.Request
	Response *gatekeeper.Response
}

type ModifyResponseResp struct {
	Response *gatekeeper.Response
	Err      *gatekeeper.Error
}

type ModifyErrorResponseArgs struct {
	Err      *gatekeeper.Error
	Request  *gatekeeper.Request
	Response *gatekeeper.Response
}

type ModifyErrorResponseResp struct {
	Response *gatekeeper.Response
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

func (c *RPCClient) ModifyRequest(req *gatekeeper.Request) (*gatekeeper.Request, *gatekeeper.Error) {
	callArgs := ModifyRequestArgs{
		Request: req,
	}
	callResp := ModifyRequestResp{}
	if err := c.client.Call("Plugin.ModifyRequest", &callArgs, &callResp); err != nil {
		return nil, gatekeeper.NewError(err)
	}
	return callResp.Request, callResp.Err
}

func (c *RPCClient) ModifyResponse(req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, *gatekeeper.Error) {
	callArgs := ModifyResponseArgs{
		Request:  req,
		Response: resp,
	}
	callResp := ModifyResponseResp{}

	if err := c.client.Call("Plugin.ModifyResponse", &callArgs, &callResp); err != nil {
		return nil, gatekeeper.NewError(err)
	}

	return callResp.Response, callResp.Err
}

func (c *RPCClient) ModifyErrorResponse(err *gatekeeper.Error, req *gatekeeper.Request, resp *gatekeeper.Response) (*gatekeeper.Response, *gatekeeper.Error) {
	callArgs := ModifyErrorResponseArgs{
		Err:      err,
		Request:  req,
		Response: resp,
	}
	callResp := ModifyErrorResponseResp{}

	if err := c.client.Call("Plugin.ModifyErrorResponse", &callArgs, &callResp); err != nil {
		return nil, gatekeeper.NewError(err)
	}
	return callResp.Response, callResp.Err
}

type RPCServer struct {
	impl   Plugin
	broker *plugin.MuxBroker

	*internal.BasePluginRPCServer
}

func (s *RPCServer) ModifyRequest(args *ModifyRequestArgs, resp *ModifyRequestResp) error {
	request, err := s.impl.ModifyRequest(args.Request)
	resp.Err = gatekeeper.NewError(err)
	resp.Request = request
	return nil
}

func (s *RPCServer) ModifyResponse(args *ModifyResponseArgs, resp *ModifyResponseResp) error {
	response, err := s.impl.ModifyResponse(args.Request, args.Response)
	resp.Err = gatekeeper.NewError(err)
	resp.Response = response
	return nil
}

func (s *RPCServer) ModifyErrorResponse(args *ModifyErrorResponseArgs, resp *ModifyErrorResponseResp) error {
	response, err := s.impl.ModifyErrorResponse(args.Err, args.Request, args.Response)
	resp.Err = gatekeeper.NewError(err)
	resp.Response = response
	return nil
}

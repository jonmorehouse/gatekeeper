package modifier

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/internal"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type modifyRequestArgs struct {
	Request *shared.Request
}
type modifyRequestResp struct {
	Request *shared.Request
	Err     *shared.Error
}

type modifyResponseArgs struct {
	Request  *shared.Request
	Response *shared.Response
}

type modifyResponseResp struct {
	Response *shared.Response
	Err      *shared.Error
}

type modifyErrorResponseArgs struct {
	Err      *shared.Error
	Request  *shared.Request
	Response *shared.Response
}

type modifyErrorResponseResp struct {
	Response *shared.Response
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

func (c *RPCClient) ModifyRequest(req *shared.Request) (*shared.Request, *shared.Error) {
	callArgs := modifyRequestArgs{
		Request: req,
	}
	callResp := modifyRequestResp{}
	if err := c.client.Call("Plugin.ModifyRequest", &callArgs, &callResp); err != nil {
		return nil, shared.NewError(err)
	}
	return callResp.Request, callResp.Err
}

func (c *RPCClient) ModifyResponse(req *shared.Request, resp *shared.Response) (*shared.Response, *shared.Error) {
	callArgs := modifyResponseArgs{
		Request:  req,
		Response: resp,
	}
	callResp := modifyResponseResp{}

	if err := c.client.Call("Plugin.ModifyResponse", &callArgs, &callResp); err != nil {
		return nil, shared.NewError(err)
	}

	return callResp.Response, callResp.Err
}

func (c *RPCClient) ModifyErrorResponse(err *shared.Error, req *shared.Request, resp *shared.Response) (*shared.Response, *shared.Error) {
	callArgs := modifyErrorResponseArgs{
		Err:      err,
		Request:  req,
		Response: resp,
	}
	callResp := modifyErrorResponseResp{}

	if err := c.client.Call("Plugin.ModifyErrorResponse", &callArgs, &callResp); err != nil {
		return nil, shared.NewError(err)
	}
	return callResp.Response, callResp.Err
}

type RPCServer struct {
	impl   Plugin
	broker *plugin.MuxBroker

	*internal.BasePluginRPCServer
}

func (s *RPCServer) ModifyRequest(args *modifyRequestArgs, resp *modifyRequestResp) error {
	request, err := s.impl.ModifyRequest(args.Request)
	resp.Err = shared.NewError(err)
	resp.Request = request
	return nil
}

func (s *RPCServer) ModifyResponse(args *modifyResponseArgs, resp *modifyResponseResp) error {
	response, err := s.impl.ModifyResponse(args.Request, args.Response)
	resp.Err = shared.NewError(err)
	resp.Response = response
	return nil
}

func (s *RPCServer) ModifyErrorResponse(args *modifyErrorResponseArgs, resp *modifyErrorResponseResp) error {
	response, err := s.impl.ModifyErrorResponse(args.Err, args.Request, args.Response)
	resp.Err = shared.NewError(err)
	resp.Response = response
	return nil
}

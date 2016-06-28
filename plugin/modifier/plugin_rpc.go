package request

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

type ModifyRequestArgs struct {
	Request *shared.Request
}
type ModifyRequestResp struct {
	Request *shared.Request
	Err     *shared.Error
}

type ModifyResponseArgs struct {
	Request  *shared.Request
	Response *shared.Response
}

type ModifyResponseResp struct {
	Response *shared.Response
	Err      *shared.Error
}

type ModifyErrorResponseArgs struct {
	Error    *shared.Error
	Request  *shared.Request
	Response *shared.Response
}

type ModifyErrorResponseResp struct {
	Response *shared.Response
	Err      *shared.Error
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

func (s *RPCServer) ModifyRequest(args *ModifyRequestArgs, resp *ModifyRequestResp) error {
	request, err := s.impl.ModifyRequest(args.Request)
	resp.Err = shared.NewError(err)
	resp.Request = request
	return nil
}

func (s *RPCServer) ModifyResponse(args *ModifyResponseArgs, resp *ModifyResponseResp) error {
	response, err := s.impl.ModifyResponse(args.Request, args.Response)
	resp.Err = shared.NewError(err)
	resp.Response = response
	return nil
}

func (s *RPCServer) ModifyErrorResponse(args *ModifyErrorResponseArgs, resp *ModifyErrorResponseResp) error {
	response, err := s.impl.ModifyErrorResponse(args.Error, args.Request, args.Response)
	resp.Err = shared.NewError(err)
	resp.Response = response
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

func (c *RPCClient) ModifyRequest(req *shared.Request) (*shared.Request, *shared.Error) {
	callArgs := ModifyRequestArgs{
		Request: req,
	}
	callResp := ModifyRequestResp{}
	if err := c.client.Call("Plugin.ModifyRequest", &callArgs, &callResp); err != nil {
		return nil, shared.NewError(err)
	}
	return callResp.Request, callResp.Err
}

func (c *RPCClient) ModifyResponse(req *shared.Request, resp *shared.Response) (*shared.Response, *shared.Error) {
	callArgs := ModifyResponseArgs{
		Request:  req,
		Response: resp,
	}
	callResp := ModifyResponseResp{}

	if err := c.client.Call("Plugin.ModifyResponse", &callArgs, &callResp); err != nil {
		return nil, shared.NewError(err)
	}

	return callResp.Response, callResp.Err
}

func (c *RPCClient) ModifyErrorResponse(err *shared.Error, req *shared.Request, resp *shared.Response) (*shared.Response, *shared.Error) {
	callArgs := ModifyErrorResponseArgs{
		Error:    err,
		Request:  req,
		Response: resp,
	}
	callResp := ModifyErrorResponseResp{}

	if err := c.client.Call("Plugin.ModifyErrorResponse", &callArgs, &callResp); err != nil {
		return nil, shared.NewError(err)
	}
	return callResp.Response, callResp.Err
}

package loadbalancer

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

type AddBackendArgs struct {
	Backend  shared.Backend
	Upstream shared.UpstreamID
}
type AddBackendResp struct {
	Err *shared.Error
}

type RemoveBackendArgs struct {
	Backend shared.Backend
}
type RemoveBackendResp struct {
	Err *shared.Error
}

type GetBackendArgs struct {
	Upstream shared.UpstreamID
}
type GetBackendResp struct {
	Backend shared.Backend
	Err     *shared.Error
}

type RPCServer struct {
	impl   Plugin
	broker *plugin.MuxBroker
}

func (s *RPCServer) Start(args *StartArgs, resp *StartResp) error {
	resp.Err = s.impl.Start()
	return nil
}

func (s *RPCServer) Stop(args *StopArgs, resp *StopResp) error {
	resp.Err = s.impl.Stop()
	return nil
}

func (s *RPCServer) Heartbeat(args *HeartbeatArgs, resp *HeartbeatResp) error {
	resp.Err = s.impl.Heartbeat()
	return nil
}

func (s *RPCServer) Configure(args *ConfigureArgs, resp *ConfigureResp) error {
	resp.Err = s.impl.Configure(args.Opts)
	return nil
}

func (s *RPCServer) AddBackend(args *AddBackendArgs, resp *AddBackendResp) error {
	resp.Err = s.impl.AddBackend(args.Upstream, args.Backend)
	return nil
}

func (s *RPCServer) RemoveBackend(args *RemoveBackendArgs, resp *RemoveBackendResp) error {
	if err := s.impl.RemoveBackend(args.Backend); err != nil {
		resp.Err = err
	}
	return nil
}

func (s *RPCServer) GetBackend(args *GetBackendArgs, resp *GetBackendResp) error {
	resp.Backend, resp.Err = s.impl.GetBackend(args.Upstream)
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

func (c *RPCClient) AddBackend(upstream shared.UpstreamID, backend shared.Backend) *shared.Error {
	callArgs := AddBackendArgs{
		Upstream: upstream,
		Backend:  backend,
	}
	callResp := AddBackendResp{}
	if err := c.client.Call("Plugin.AddBackend", &callArgs, &callResp); err != nil {
		return shared.NewError(err)
	}
	return callResp.Err
}

func (c *RPCClient) RemoveBackend(backend shared.Backend) *shared.Error {
	callArgs := RemoveBackendArgs{
		Backend: backend,
	}
	callResp := RemoveBackendResp{}
	if err := c.client.Call("Plugin.RemoveBackend", &callArgs, &callResp); err != nil {
		return shared.NewError(err)
	}
	return callResp.Err
}

func (c *RPCClient) GetBackend(upstream shared.UpstreamID) (shared.Backend, *shared.Error) {
	callArgs := GetBackendArgs{
		Upstream: upstream,
	}
	callResp := GetBackendResp{}
	if err := c.client.Call("Plugin.GetBackend", &callArgs, &callResp); err != nil {
		return shared.NilBackend, shared.NewError(err)
	}
	return callResp.Backend, callResp.Err
}

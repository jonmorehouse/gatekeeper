package loadbalancer

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type StartArgs struct{}
type StartResp struct {
	Err error
}

type StopArgs struct{}
type StopResp struct {
	Err error
}

type ConfigureArgs struct {
	Opts map[string]interface{}
}
type ConfigureResp struct {
	Err error
}

type HeartbeatArgs struct{}
type HeartbeatResp struct {
	Err error
}

type AddBackendArgs struct {
	Backend  shared.Backend
	Upstream shared.UpstreamID
}
type AddBackendResp struct {
	Err error
}

type RemoveBackendArgs struct {
	Backend shared.Backend
}
type RemoveBackendResp struct {
	Err error
}

type GetBackendArgs struct {
	Upstream shared.UpstreamID
}
type GetBackendResp struct {
	Backend shared.Backend
	Err     error
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

func (c *RPCClient) Start() error {
	callArgs := StartArgs{}
	callResp := StartResp{}
	if err := c.client.Call("Plugin.Start", &callArgs, &callResp); err != nil {
		return err
	}
	return callResp.Err
}

func (c *RPCClient) Stop() error {
	callArgs := StopArgs{}
	callResp := StopResp{}
	if err := c.client.Call("Plugin.Stop", &callArgs, &callResp); err != nil {
		return err
	}
	return callResp.Err
}

func (c *RPCClient) Heartbeat() error {
	callArgs := HeartbeatArgs{}
	callResp := HeartbeatResp{}
	if err := c.client.Call("Plugin.Heartbeat", &callArgs, &callResp); err != nil {
		return err
	}
	return callResp.Err
}

func (c *RPCClient) Configure(opts map[string]interface{}) error {
	callArgs := ConfigureArgs{
		Opts: opts,
	}
	callResp := ConfigureResp{}
	if err := c.client.Call("Plugin.Configure", &callArgs, &callResp); err != nil {
		return err
	}
	return callResp.Err
}

func (c *RPCClient) AddBackend(upstream shared.UpstreamID, backend shared.Backend) error {
	callArgs := AddBackendArgs{
		Upstream: upstream,
		Backend:  backend,
	}
	callResp := AddBackendResp{}
	if err := c.client.Call("Plugin.AddBackend", &callArgs, &callResp); err != nil {
		return err
	}
	return callResp.Err
}

func (c *RPCClient) RemoveBackend(backend shared.Backend) error {
	callArgs := RemoveBackendArgs{
		Backend: backend,
	}
	callResp := RemoveBackendResp{}
	if err := c.client.Call("Plugin.RemoveBackend", &callArgs, &callResp); err != nil {
		return err
	}
	return callResp.Err
}

func (c *RPCClient) GetBackend(upstream shared.UpstreamID) (shared.Backend, error) {
	callArgs := GetBackendArgs{
		Upstream: upstream,
	}
	callResp := GetBackendResp{}
	if err := c.client.Call("plugin.GetBackend", &callArgs, &callResp); err != nil {
		return shared.NilBackend, callResp.Err
	}
	return callResp.Backend, callResp.Err
}

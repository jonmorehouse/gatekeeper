package upstream

import (
	"net/rpc"

	"github.com/jonmorehouse/gatekeeper/shared"
)

type AddUpstreamArgs struct {
	Upstream shared.Upstream
}

type AddUpstreamResp struct {
	Err error
}

type RemoveUpstreamArgs struct {
	UpstreamID shared.UpstreamID
}

type RemoveUpstreamResp struct {
	Err error
}

type AddBackendArgs struct {
	UpstreamID shared.UpstreamID
	Backend    shared.Backend
}

type AddBackendResp struct {
	Err error
}

type RemoveBackendArgs struct {
	BackendID shared.BackendID
}

type RemoveBackendResp struct {
	Err error
}

type HeartbeatArgs struct{}
type HeartbeatResp struct {
	Err error
}

type ManagerRPCClient struct {
	client *rpc.Client
}

func (c *ManagerRPCClient) Notify() error {
	return c.client.Call("Plugin.Notify", new(interface{}), new(interface{}))
}

func (c *ManagerRPCClient) Heartbeat() error {
	callArgs := HeartbeatArgs{}
	callResp := HeartbeatResp{}

	if err := c.client.Call("Plugin.Heartbeat", &callArgs, &callResp); err != nil {
		return err
	}
	return callResp.Err
}

func (c *ManagerRPCClient) AddUpstream(upstream shared.Upstream) error {
	callArgs := AddUpstreamArgs{
		Upstream: upstream,
	}
	callResp := AddUpstreamResp{}

	if err := c.client.Call("Plugin.AddUpstream", &callArgs, &callResp); err != nil {
		return err
	}

	return callResp.Err
}

func (c *ManagerRPCClient) RemoveUpstream(upstreamID shared.UpstreamID) error {
	callArgs := RemoveUpstreamArgs{
		UpstreamID: upstreamID,
	}
	callResp := RemoveUpstreamResp{}

	if err := c.client.Call("Plugin.RemoveUpstream", &callArgs, &callResp); err != nil {
		return err
	}
	return callResp.Err
}

func (c *ManagerRPCClient) AddBackend(upstreamID shared.UpstreamID, backend shared.Backend) error {
	callArgs := AddBackendArgs{
		UpstreamID: upstreamID,
		Backend:    backend,
	}
	callResp := AddBackendResp{}

	if err := c.client.Call("Plugin.AddBackend", &callArgs, &callResp); err != nil {
		return err
	}
	return callResp.Err
}

func (c *ManagerRPCClient) RemoveBackend(backendID shared.BackendID) error {
	callArgs := RemoveBackendArgs{
		BackendID: backendID,
	}
	callResp := RemoveBackendResp{}

	if err := c.client.Call("Plugin.RemoveBackend", &callArgs, &callResp); err != nil {
		return err
	}
	return callResp.Err
}

type ManagerRPCServer struct {
	impl        Manager
	connectedCh chan interface{}
}

func (s *ManagerRPCServer) Notify(*interface{}, *interface{}) error {
	s.connectedCh <- new(interface{})
	return nil
}

func (s *ManagerRPCServer) Heartbeat(args *HeartbeatArgs, resp *HeartbeatResp) error {
	return nil
}

func (s *ManagerRPCServer) AddUpstream(args *AddUpstreamArgs, resp *AddUpstreamResp) error {
	err := s.impl.AddUpstream(args.Upstream)
	resp.Err = err
	return nil
}

func (s *ManagerRPCServer) RemoveUpstream(args *RemoveUpstreamArgs, resp *RemoveUpstreamResp) error {
	err := s.impl.RemoveUpstream(args.UpstreamID)
	resp.Err = err
	return nil
}

func (s *ManagerRPCServer) AddBackend(args *AddBackendArgs, resp *AddBackendResp) error {
	err := s.impl.AddBackend(args.UpstreamID, args.Backend)
	resp.Err = err
	return nil
}

func (s *ManagerRPCServer) RemoveBackend(args *RemoveBackendArgs, resp *RemoveBackendResp) error {
	err := s.impl.RemoveBackend(args.BackendID)
	resp.Err = err
	return nil
}

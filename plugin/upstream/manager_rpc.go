package upstream

import (
	"net/rpc"

	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type NotifyArgs struct{}
type NotifyResp struct{}

type AddUpstreamArgs struct {
	Upstream *gatekeeper.Upstream
}

type AddUpstreamResp struct {
	Err *gatekeeper.Error
}

type RemoveUpstreamArgs struct {
	UpstreamID gatekeeper.UpstreamID
}

type RemoveUpstreamResp struct {
	Err *gatekeeper.Error
}

type AddBackendArgs struct {
	UpstreamID gatekeeper.UpstreamID
	Backend    *gatekeeper.Backend
}

type AddBackendResp struct {
	Err *gatekeeper.Error
}

type RemoveBackendArgs struct {
	BackendID gatekeeper.BackendID
}

type RemoveBackendResp struct {
	Err *gatekeeper.Error
}

type HeartbeatArgs struct{}
type HeartbeatResp struct {
	Err *gatekeeper.Error
}

type ManagerRPCClient struct {
	client *rpc.Client
}

func (c *ManagerRPCClient) Close() {
	c.client.Close()
}

func (c *ManagerRPCClient) Notify() *gatekeeper.Error {
	err := c.client.Call("Plugin.Notify", &NotifyArgs{}, &NotifyResp{})
	return gatekeeper.NewError(err)
}

func (c *ManagerRPCClient) Heartbeat() *gatekeeper.Error {
	callArgs := HeartbeatArgs{}
	callResp := HeartbeatResp{}

	if err := c.client.Call("Plugin.Heartbeat", &callArgs, &callResp); err != nil {
		return gatekeeper.NewError(err)
	}
	return callResp.Err
}

func (c *ManagerRPCClient) AddUpstream(upstream *gatekeeper.Upstream) *gatekeeper.Error {
	callArgs := AddUpstreamArgs{
		Upstream: upstream,
	}
	callResp := AddUpstreamResp{}

	if err := c.client.Call("Plugin.AddUpstream", &callArgs, &callResp); err != nil {
		return gatekeeper.NewError(err)
	}

	return callResp.Err
}

func (c *ManagerRPCClient) RemoveUpstream(upstreamID gatekeeper.UpstreamID) *gatekeeper.Error {
	callArgs := RemoveUpstreamArgs{
		UpstreamID: upstreamID,
	}
	callResp := RemoveUpstreamResp{}

	if err := c.client.Call("Plugin.RemoveUpstream", &callArgs, &callResp); err != nil {
		return gatekeeper.NewError(err)
	}
	return callResp.Err
}

func (c *ManagerRPCClient) AddBackend(upstreamID gatekeeper.UpstreamID, backend *gatekeeper.Backend) *gatekeeper.Error {
	callArgs := AddBackendArgs{
		UpstreamID: upstreamID,
		Backend:    backend,
	}
	callResp := AddBackendResp{}

	if err := c.client.Call("Plugin.AddBackend", &callArgs, &callResp); err != nil {
		return gatekeeper.NewError(err)
	}
	return callResp.Err
}

func (c *ManagerRPCClient) RemoveBackend(backendID gatekeeper.BackendID) *gatekeeper.Error {
	callArgs := RemoveBackendArgs{
		BackendID: backendID,
	}
	callResp := RemoveBackendResp{}

	if err := c.client.Call("Plugin.RemoveBackend", &callArgs, &callResp); err != nil {
		return gatekeeper.NewError(err)
	}
	return callResp.Err
}

type ManagerRPCServer struct {
	impl        Manager
	connectedCh chan struct{}
}

func (s *ManagerRPCServer) Notify(*NotifyArgs, *NotifyResp) error {
	s.connectedCh <- struct{}{}
	return nil
}

func (s *ManagerRPCServer) Heartbeat(args *HeartbeatArgs, resp *HeartbeatResp) error {
	return nil
}

func (s *ManagerRPCServer) AddUpstream(args *AddUpstreamArgs, resp *AddUpstreamResp) error {
	err := s.impl.AddUpstream(args.Upstream)
	resp.Err = gatekeeper.NewError(err)
	return nil
}

func (s *ManagerRPCServer) RemoveUpstream(args *RemoveUpstreamArgs, resp *RemoveUpstreamResp) error {
	err := s.impl.RemoveUpstream(args.UpstreamID)
	resp.Err = gatekeeper.NewError(err)
	return nil
}

func (s *ManagerRPCServer) AddBackend(args *AddBackendArgs, resp *AddBackendResp) error {
	err := s.impl.AddBackend(args.UpstreamID, args.Backend)
	resp.Err = gatekeeper.NewError(err)
	return nil
}

func (s *ManagerRPCServer) RemoveBackend(args *RemoveBackendArgs, resp *RemoveBackendResp) error {
	err := s.impl.RemoveBackend(args.BackendID)
	resp.Err = gatekeeper.NewError(err)
	return nil
}

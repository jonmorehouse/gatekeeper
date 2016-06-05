package upstream

import "net/rpc"

type AddUpstreamArgs struct {
	Upstream Upstream
}

type AddUpstreamResp struct {
	UpstreamID UpstreamID
	Err        error
}

type RemoveUpstreamArgs struct {
	UpstreamID UpstreamID
}

type RemoveUpstreamResp struct {
	Err error
}

type AddBackendArgs struct {
	UpstreamID UpstreamID
	Backend    Backend
}

type AddBackendResp struct {
	BackendID BackendID
	Err       error
}

type RemoveBackendArgs struct {
	BackendID BackendID
}

type RemoveBackendResp struct {
	Err error
}

type HeartbeatArgs struct{}
type HeartbeatResp struct{}

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

func (c *ManagerRPCClient) AddUpstream(upstream Upstream) (UpstreamID, error) {
	callArgs := AddUpstreamArgs{
		Upstream: upstream,
	}
	callResp := AddUpstreamResp{}

	if err := c.client.Call("Plugin.AddUpstream", &callArgs, &callResp); err != nil {
		return NilUpstreamID, err
	}

	return callResp.UpstreamID, callResp.Err
}

func (c *ManagerRPCClient) RemoveUpstream(upstreamID UpstreamID) error {
	callArgs := RemoveUpstreamArgs{
		UpstreamID: upstreamID,
	}
	callResp := RemoveUpstreamResp{}

	if err := c.client.Call("Plugin.RemoveUpstream", &callArgs, &callResp); err != nil {
		return err
	}
	return callResp.Err
}

func (c *ManagerRPCClient) AddBackend(upstreamID UpstreamID, backend Backend) (BackendID, error) {
	callArgs := AddBackendArgs{
		UpstreamID: upstreamID,
		Backend:    backend,
	}
	callResp := AddBackendResp{}

	if err := c.client.Call("Plugin.AddBackend", &callArgs, &callResp); err != nil {
		return NilBackendID, err
	}
	return callResp.BackendID, callResp.Err
}

func (c *ManagerRPCClient) RemoveBackend(backendID BackendID) error {
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
	upstreamID, err := s.impl.AddUpstream(args.Upstream)
	resp.UpstreamID = upstreamID
	resp.Err = err
	return nil
}

func (s *ManagerRPCServer) RemoveUpstream(args *RemoveUpstreamArgs, resp *RemoveUpstreamResp) error {
	err := s.impl.RemoveUpstream(args.UpstreamID)
	resp.Err = err
	return nil
}

func (s *ManagerRPCServer) AddBackend(args *AddBackendArgs, resp *AddBackendResp) error {
	backendID, err := s.impl.AddBackend(args.UpstreamID, args.Backend)
	resp.Err = err
	resp.BackendID = backendID
	return nil
}

func (s *ManagerRPCServer) RemoveBackend(args *RemoveBackendArgs, resp *RemoveBackendResp) error {
	err := s.impl.RemoveBackend(args.BackendID)
	resp.Err = err
	return nil
}

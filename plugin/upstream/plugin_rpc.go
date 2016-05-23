package upstream

import (
	"fmt"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

type ConfigureArgs struct {
	Opts Opts
}
type ConfigureResp struct {
	Error error
}

type FetchUpstreamsArgs struct{}
type FetchUpstreamsResp struct {
	Upstreams []Upstream
	Error     error
}

type FetchUpstreamBackendsArgs struct {
	UpstreamID UpstreamID
}

type FetchUpstreamBackendsResp struct {
	Backends []Backend
	Error    error
}

type StartArgs struct {
	// the connection id to accept on the parent's side of things!
	ConnID uint32
}
type StartResp struct {
	Error error
}

type StopArgs struct{}
type StopResp struct {
	Errors []error
}

// this is what the plugin is actually running behind the scenes
type PluginRPCServer struct {
	// this is the public Plugin that the user has created
	plugin  Plugin
	broker  *plugin.MuxBroker
	manager Manager
}

func (s *PluginRPCServer) Configure(args *ConfigureArgs, resp *ConfigureResp) error {
	if err := s.plugin.Configure(args.Opts); err != nil {
		resp.Error = err
	}
	return nil
}

func (s *PluginRPCServer) FetchUpstreams(args *FetchUpstreamsArgs, resp *FetchUpstreamsResp) error {
	upstreams, err := s.plugin.FetchUpstreams()
	resp.Upstreams = upstreams
	resp.Error = err

	return nil
}

func (s *PluginRPCServer) FetchUpstreamBackends(args *FetchUpstreamBackendsArgs, resp *FetchUpstreamBackendsResp) error {
	backends, err := s.plugin.FetchUpstreamBackends(args.UpstreamID)
	resp.Backends = backends
	resp.Error = err

	return nil
}

func (s *PluginRPCServer) Start(args *StartArgs, resp *StartResp) error {
	if s.manager != nil {
		return fmt.Errorf("Manager already started; must stop first")
	}

	conn, err := s.broker.Dial(args.ConnID)
	if err != nil {
		return err
	}

	// create an RPC connection to the parent's upstream manager and pass it into the plugin userspace
	client := rpc.NewClient(conn)
	manager := &ManagerRPCClient{
		client: client,
	}
	// we call notify, so that the parent manager server can verify a
	// connection was made. This is done in the background, and the manager
	// server listens until it hears this call.
	go func() {
		manager.Notify()
	}()
	// Notify is not availably on the Manager interface and should not be called outside of any other context
	s.manager = manager

	if err := s.plugin.Start(s.manager); err != nil {
		resp.Error = err
	}
	return nil
}

func (s *PluginRPCServer) Stop(args *StopArgs, resp *StopResp) error {
	if s.manager == nil {
		return fmt.Errorf("Manager not started...")
	}

	// if the plugin stop fails, we pass that along upstream, but still try
	// to make sure the connection is closed
	if err := s.plugin.Stop(); err != nil {
		resp.Errors = append(resp.Errors, err)
	}

	// the manager owns its connection to the RPCServer, we go ahead and
	// try to close it. If it errs out, we actually care about it at the RPC level
	if err := s.plugin.Stop(); err != nil {
		resp.Errors = append(resp.Errors, err)
	}

	return nil
}

// this implements the RPC that gatekeeper will interact with ...
type PluginRPCClient struct {
	broker *plugin.MuxBroker
	client *rpc.Client
}

func (c *PluginRPCClient) Configure(opts Opts) error {
	callArgs := ConfigureArgs{
		Opts: opts,
	}
	callResp := ConfigureResp{}

	if err := c.client.Call("Plugin.Configure", &callArgs, &callResp); err != nil {
		return err
	}

	return callResp.Error
}

func (c *PluginRPCClient) FetchUpstreams() ([]Upstream, error) {
	callArgs := FetchUpstreamsArgs{}
	callResp := FetchUpstreamsResp{}

	if err := c.client.Call("Plugin.FetchUpstreams", &callArgs, &callResp); err != nil {
		return []Upstream{}, err
	}

	return callResp.Upstreams, callResp.Error
}

func (c *PluginRPCClient) FetchUpstreamBackends(upstreamID UpstreamID) ([]Backend, error) {
	callArgs := FetchUpstreamBackendsArgs{
		UpstreamID: upstreamID,
	}
	callResp := FetchUpstreamBackendsResp{}

	if err := c.client.Call("Plugin.FetchUpstreamBackends", &callArgs, &callResp); err != nil {
		return []Backend{}, err
	}

	return callResp.Backends, callResp.Error
}

func (c *PluginRPCClient) Start(manager Manager) error {
	connID := c.broker.NextId()
	callArgs := StartArgs{connID}
	callResp := StartResp{}

	// start a server and run it in a goroutine; this will accept rpc calls
	// from the child and run them in the correct place
	connectedCh := make(chan interface{})
	go func() {
		managerRPCServer := ManagerRPCServer{
			impl:        manager,
			connectedCh: connectedCh,
		}
		c.broker.AcceptAndServe(connID, &managerRPCServer)
	}()

	if err := c.client.Call("Plugin.Start", &callArgs, &callResp); err != nil {
		return err
	}

	// before returning, we ensure that the plugin has verified
	// connectivity with our managerRPCServer. Specifically, we don't care
	// if this is done before or after they run start, just so long as we
	// verify connectivity before returning here.
	<-connectedCh
	close(connectedCh)

	return nil
}

func (c *PluginRPCClient) Stop() error {
	callArgs := StopArgs{}
	callResp := StopResp{}
	if err := c.client.Call("Plugin.Stop", &callArgs, &callResp); err != nil {
		return err
	}

	c.client.Close()
	return nil
}

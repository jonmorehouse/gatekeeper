package upstream

import (
	"fmt"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/shared"
)

type ConfigureArgs struct {
	Opts map[string]interface{}
}
type ConfigureResp struct {
	Err *shared.Error
}

type StartArgs struct {
	// the connection id to accept on the parent's side of things!
	ConnID uint32
}
type StartResp struct {
	Err *shared.Error
}

type StopArgs struct{}
type StopResp struct {
	Err *shared.Error
}

// this is what the plugin is actually running behind the scenes
type PluginRPCServer struct {
	// impl is what Implements the public PluginRPC interface
	impl   Plugin
	broker *plugin.MuxBroker

	// ManagerRPC is an interface wrapping Manager over RPC
	managerRPC ManagerRPC
}

func (s *PluginRPCServer) Configure(args *ConfigureArgs, resp *ConfigureResp) error {
	if err := s.impl.Configure(args.Opts); err != nil {
		resp.Err = shared.NewError(err)
	}
	return nil
}

func (s *PluginRPCServer) Heartbeat(args *HeartbeatArgs, resp *HeartbeatResp) error {
	if err := s.managerRPC.Heartbeat(); err != nil {
		return shared.NewError(err)
	}

	return s.impl.Heartbeat()
}

func (s *PluginRPCServer) Start(args *StartArgs, resp *StartResp) error {
	if s.managerRPC != nil {
		return shared.NewError(fmt.Errorf("Manager already started; must stop first"))
	}

	conn, err := s.broker.Dial(args.ConnID)
	if err != nil {
		return shared.NewError(err)
	}

	// create an RPC connection back to the parent, connecting againt the
	// ManagerRPC interface that is being served from the parent process.
	client := rpc.NewClient(conn)

	// ManagerRPC implements the ManagerRPC interface, wrapping the Manager
	// interface that talks back to the parent over RPC.
	managerRPC := &ManagerRPCClient{client: client}

	// because ManagerRPC returns concrete *shared.Error types, we wrap it
	// in layer to make it a nicer experience on the "Plugin" side.
	// Specifically, this simply returns straight error types instead of
	// *shared.Error types

	// NOTE: we call Notify() to inform the parent process which is serving
	// this manager over RPC that we have successfully connected from the
	// plugin process. This method is no longer available once we downcast
	// this type to a ManagerRPC by adding it to the current type.
	go managerRPC.Notify()
	s.managerRPC = managerRPC

	// create a ManagerClient which is a wrapper around ManagerRPC which is
	// what is passed along to the plugin's implementer.
	managerWrapper := &ManagerClient{ManagerRPC: managerRPC}
	if err := s.impl.Start(managerWrapper); err != nil {
		resp.Err = shared.NewError(err)
	}

	return nil
}

func (s *PluginRPCServer) Stop(args *StopArgs, resp *StopResp) error {
	if s.managerRPC == nil {
		resp.Err = shared.NewError(fmt.Errorf("No manager configured"))
		return nil
	}

	// if the plugin stop fails, we pass that along upstream, but still try
	// to make sure the connection is closed
	if err := s.impl.Stop(); err != nil {
		resp.Err = shared.NewError(err)
		return nil
	}

	// the manager owns its connection to the RPCServer, we go ahead and
	// try to close it. If it errs out, we actually care about it at the RPC level
	managerRPCClient, ok := s.managerRPC.(*ManagerRPCClient)
	if !ok {
		return fmt.Errorf("invalid managerRPCClient; this is a gatekeeper-internal error")
	}
	managerRPCClient.Close()
	return nil
}

// this implements the RPC that gatekeeper will interact with ...
type PluginRPCClient struct {
	broker *plugin.MuxBroker
	client *rpc.Client

	// manager is the local type, in the parent process, which implements
	// the upstream_plugin.Manager interface. Specifically, this is passed
	// along and built into a ManagerRPC that is run inside the parent
	// process accepting requests from plugins calling back into the
	// parent.
	manager Manager
}

// Configure the client and the server. Specifically, this requires that a Manager is passed in here
func (c *PluginRPCClient) Configure(opts map[string]interface{}) *shared.Error {
	rawManager, ok := opts["manager"]
	if !ok {
		return shared.NewError(fmt.Errorf("No manager was passed into the rpc client"))
	}

	if manager, ok := rawManager.(Manager); !ok {
		return shared.NewError(fmt.Errorf("Manager was passed in, but it was not successfully cast to a Manager type"))
	} else {
		c.manager = manager
		delete(opts, "manager")
	}

	callArgs := ConfigureArgs{
		Opts: opts,
	}
	callResp := ConfigureResp{}

	if err := c.client.Call("Plugin.Configure", &callArgs, &callResp); err != nil {
		return shared.NewError(err)
	}

	return callResp.Err
}

func (c *PluginRPCClient) Heartbeat() *shared.Error {
	callArgs := HeartbeatArgs{}
	callResp := HeartbeatResp{}

	if err := c.client.Call("Plugin.Heartbeat", &callArgs, &callResp); err != nil {
		return shared.NewError(err)
	}

	return callResp.Err
}

func (c *PluginRPCClient) Start() *shared.Error {
	connID := c.broker.NextId()
	callArgs := StartArgs{connID}
	callResp := StartResp{}

	// Start a ManagerRPCServer, which will take the impl, passing methods
	// along to it and ensuring that the correct types are passed around in
	// response.
	connectedCh := make(chan struct{})
	go func() {
		managerRPCServer := ManagerRPCServer{
			impl:        c.manager,
			connectedCh: connectedCh,
		}
		c.broker.AcceptAndServe(connID, &managerRPCServer)
	}()

	if err := c.client.Call("Plugin.Start", &callArgs, &callResp); err != nil {
		return shared.NewError(err)
	}

	// before returning, we ensure that the plugin has verified
	// connectivity with our managerRPCServer. Specifically, we don't care
	// if this is done before or after they run start, just so long as we
	// verify connectivity before returning here.
	<-connectedCh
	close(connectedCh)
	return callResp.Err
}

func (c *PluginRPCClient) Stop() *shared.Error {
	callArgs := StopArgs{}
	callResp := StopResp{}
	if err := c.client.Call("Plugin.Stop", &callArgs, &callResp); err != nil {
		return shared.NewError(err)
	}

	c.client.Close()
	return callResp.Err
}

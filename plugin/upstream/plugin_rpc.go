package upstream

import (
	"fmt"
	"log"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
)

type ConfigureArgs struct {
	Opts map[string]interface{}
}
type ConfigureResp struct {
	Err error
}

type StartArgs struct {
	// the connection id to accept on the parent's side of things!
	ConnID uint32
}
type StartResp struct {
	Err error
}

type StopArgs struct{}
type StopResp struct {
	Err error
}

// this is what the plugin is actually running behind the scenes
type PluginRPCServer struct {
	// impl is what Implements the public Plugin interface
	impl    Plugin
	broker  *plugin.MuxBroker
	manager Manager
}

func (s *PluginRPCServer) Configure(args *ConfigureArgs, resp *ConfigureResp) error {
	log.Println("RPC server configured")
	if err := s.impl.Configure(args.Opts); err != nil {
		resp.Err = err
	}
	return nil
}

func (s *PluginRPCServer) Heartbeat(args *HeartbeatArgs, resp *HeartbeatResp) error {
	rpcManager, ok := s.manager.(*ManagerRPCClient)
	if !ok {
		return fmt.Errorf("Fatal: upstreams plugin's RPCServer has an RPCClient with no HeartBeat method")
	}

	if err := rpcManager.Heartbeat(); err != nil {
		return err
	}

	return s.impl.Heartbeat()
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
	go manager.Notify()

	// Notify is not availably on the Manager interface and should not be called from any other context
	s.manager = manager
	if err := s.impl.Start(s.manager); err != nil {
		resp.Err = err
	}

	return nil
}

func (s *PluginRPCServer) Stop(args *StopArgs, resp *StopResp) error {
	if s.manager == nil {
		resp.Err = fmt.Errorf("No manager configured")
		return nil
	}

	// if the plugin stop fails, we pass that along upstream, but still try
	// to make sure the connection is closed
	if err := s.impl.Stop(); err != nil {
		resp.Err = err
		return nil
	}

	// the manager owns its connection to the RPCServer, we go ahead and
	// try to close it. If it errs out, we actually care about it at the RPC level
	if err := s.impl.Stop(); err != nil {
		resp.Err = err
	}

	return nil
}

// this implements the RPC that gatekeeper will interact with ...
type PluginRPCClient struct {
	broker  *plugin.MuxBroker
	client  *rpc.Client
	manager Manager
}

// Configure the client and the server. Specifically, this requires that a Manager is passed in here
func (c *PluginRPCClient) Configure(opts map[string]interface{}) error {
	rawManager, ok := opts["manager"]
	if !ok {
		return fmt.Errorf("No manager was passed into the rpc client")
	}

	if manager, ok := rawManager.(Manager); !ok {
		return fmt.Errorf("Manager was passed in, but it was not successfully cast to a Manager type")
	} else {
		c.manager = manager
		delete(opts, "manager")
	}

	callArgs := ConfigureArgs{
		Opts: opts,
	}
	callResp := ConfigureResp{}

	if err := c.client.Call("Plugin.Configure", &callArgs, &callResp); err != nil {
		return err
	}

	return callResp.Err
}

func (c *PluginRPCClient) Heartbeat() error {
	callArgs := HeartbeatArgs{}
	callResp := HeartbeatResp{}

	if err := c.client.Call("Plugin.Heartbeat", &callArgs, &callResp); err != nil {
		return err
	}

	return callResp.Err
}

func (c *PluginRPCClient) Start() error {
	connID := c.broker.NextId()
	callArgs := StartArgs{connID}
	callResp := StartResp{}

	// start a server and run it in a goroutine; this will accept rpc calls
	// from the child and run them in the correct place
	connectedCh := make(chan interface{})
	go func() {
		managerRPCServer := ManagerRPCServer{
			impl:        c.manager,
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

	return callResp.Err
}

func (c *PluginRPCClient) Stop() error {
	callArgs := StopArgs{}
	callResp := StopResp{}
	if err := c.client.Call("Plugin.Stop", &callArgs, &callResp); err != nil {
		return err
	}

	c.client.Close()
	return callResp.Err
}

package upstream

import (
	"fmt"
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/internal"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

type setManagerArgs struct {
	ConnID uint32
}

type setManagerResp struct {
	Err *gatekeeper.Error
}

type upstreamMetricArgs struct {
	Metrics []*gatekeeper.UpstreamMetric
}
type upstreamMetricResp struct {
	Errs []*gatekeeper.Error
}

// PluginRPC is a representation of the Plugin interface that is RPC safe. It
// embeds an internal.BasePluginRPC which handles the basic RPC client
// communications of the `Start`, `Stop`, `Configure` and `Heartbeat` methods.
type RPCClient struct {
	broker *plugin.MuxBroker
	client *rpc.Client

	*internal.BasePluginRPCClient
}

func (c *RPCClient) SetManager(manager Manager) *gatekeeper.Error {
	connID := c.broker.NextId()

	// Start a ManagerRPCServer, which will take the impl, passing methods
	// along to it and ensuring that the correct types are passed around in
	// response.
	connectedCh := make(chan struct{})
	go func() {
		managerRPCServer := ManagerRPCServer{
			impl:        manager,
			connectedCh: connectedCh,
		}
		c.broker.AcceptAndServe(connID, &managerRPCServer)
	}()

	args := &setManagerArgs{connID}
	resp := &setManagerResp{}

	if err := c.client.Call("Plugin.SetManager", args, resp); err != nil {
		return gatekeeper.NewError(err)
	}

	// before returning, we ensure that the plugin has verified
	// connectivity with our managerRPCServer. Specifically, we don't care
	// if this is done before or after they run start, just so long as we
	// verify connectivity before returning here.
	<-connectedCh
	close(connectedCh)
	return resp.Err
}

func (c *RPCClient) UpstreamMetric(metrics []*gatekeeper.UpstreamMetric) []*gatekeeper.Error {
	callArgs := upstreamMetricArgs{
		Metrics: metrics,
	}
	callResp := upstreamMetricResp{}

	if err := c.client.Call("Plugin.UpstreamMetric", &callArgs, &callResp); err != nil {
		return []*gatekeeper.Error{gatekeeper.NewError(err)}
	}

	return callResp.Errs
}

type RPCServer struct {
	impl   Plugin
	broker *plugin.MuxBroker

	*internal.BasePluginRPCServer

	// This is created from the SetManager method
	managerRPC ManagerRPC
}

func (s *RPCServer) Stop(args *internal.StopArgs, resp *internal.StopResp) error {
	managerClient, ok := s.managerRPC.(*ManagerRPCClient)
	if ok {
		managerClient.Close()
	}

	return s.BasePluginRPCServer.Stop(args, resp)
}

func (s *RPCServer) SetManager(args *setManagerArgs, resp *setManagerResp) error {
	if s.managerRPC != nil {
		return gatekeeper.NewError(fmt.Errorf("Manager already started; must stop first"))
	}

	conn, err := s.broker.Dial(args.ConnID)
	if err != nil {
		return gatekeeper.NewError(err)
	}

	// create an RPC connection back to the parent, connecting againt the
	// ManagerRPC interface that is being served from the parent process.
	client := rpc.NewClient(conn)

	// ManagerRPC implements the ManagerRPC interface, wrapping the Manager
	// interface that talks back to the parent over RPC.
	managerRPC := &ManagerRPCClient{client: client}

	// NOTE: we call Notify() to inform the parent process which is serving
	// this manager over RPC that we have successfully connected from the
	// plugin process. This method is no longer available once we downcast
	// this type to a ManagerRPC by adding it to the current type.
	go managerRPC.Notify()
	s.managerRPC = managerRPC

	// create a ManagerClient which is a wrapper around ManagerRPC which is
	// what is passed along to the plugin implementation. This implements
	// the Manager interface, not the ManagerRPC interface
	return s.impl.SetManager(&ManagerClient{ManagerRPC: managerRPC})
}

func (s *RPCServer) UpstreamMetric(args *upstreamMetricArgs, resp *upstreamMetricResp) error {
	errs := make([]*gatekeeper.Error, 0, len(args.Metrics))
	for _, metric := range args.Metrics {
		if err := s.impl.UpstreamMetric(metric); err != nil {
			errs = append(errs, gatekeeper.NewError(err))
		}
	}

	resp.Errs = errs
	return nil
}

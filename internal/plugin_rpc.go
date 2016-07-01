package internal

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

type HeartbeatArgs struct{}
type HeartbeatResp struct {
	Err *shared.Error
}

type ConfigureArgs struct {
	Args map[string]interface{}
}
type ConfigureResp struct {
	Err *shared.Error
}

type BasePluginRPCClient struct {
	client *rpc.Client
	broker *plugin.MuxBroker
}

func NewBasePluginRPCClient(broker *plugin.MuxBroker, client *rpc.Client) *BasePluginRPCClient {
	return &BasePluginRPCClient{
		client: client,
		broker: broker,
	}
}

func (c *BasePluginRPCClient) Start() *shared.Error {
	args := &StartArgs{}
	resp := &StartResp{}

	if err := c.client.Call("Plugin.Start", args, resp); err != nil {
		return shared.NewError(err)
	}

	return resp.Err
}

func (c *BasePluginRPCClient) Stop() *shared.Error {
	args := &StopArgs{}
	resp := &StopResp{}

	if err := c.client.Call("Plugin.Stop", args, resp); err != nil {
		return shared.NewError(err)
	}

	return resp.Err
}

func (c *BasePluginRPCClient) Heartbeat() *shared.Error {
	args := &HeartbeatArgs{}
	resp := &HeartbeatResp{}

	if err := c.client.Call("Plugin.Heartbeat", args, resp); err != nil {
		return shared.NewError(err)
	}

	return resp.Err
}

func (c *BasePluginRPCClient) Configure(data map[string]interface{}) *shared.Error {
	args := &ConfigureArgs{
		Args: data,
	}
	resp := &ConfigureResp{}

	if err := c.client.Call("Plugin.Configure", args, resp); err != nil {
		return shared.NewError(err)
	}

	return resp.Err
}

func NewBasePluginRPCServer(broker *plugin.MuxBroker, impl BasePlugin) *BasePluginRPCServer {
	return &BasePluginRPCServer{
		broker: broker,
		impl:   impl,
	}
}

type BasePluginRPCServer struct {
	broker *plugin.MuxBroker
	impl   BasePlugin
}

func (b *BasePluginRPCServer) Start(args *StartArgs, resp *StartResp) error {
	if err := b.impl.Start(); err != nil {
		resp.Err = shared.NewError(err)
	}

	return nil
}

func (b *BasePluginRPCServer) Stop(args *StopArgs, resp *StopResp) error {
	if err := b.impl.Stop(); err != nil {
		resp.Err = shared.NewError(err)
	}
	return nil
}

func (b *BasePluginRPCServer) Heartbeat(args *HeartbeatArgs, resp *HeartbeatResp) error {
	if err := b.impl.Heartbeat(); err != nil {
		resp.Err = shared.NewError(err)
	}
	return nil
}

func (b *BasePluginRPCServer) Configure(args *ConfigureArgs, resp *ConfigureResp) error {
	if err := b.impl.Configure(args.Args); err != nil {
		resp.Err = shared.NewError(err)
	}
	return nil
}

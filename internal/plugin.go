package internal

import (
	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/gatekeeper"
)

// BasePlugin is the most general implementation that any plugin must
// implement. Specifically, this is used internally in gatekeeper/core to
// handle some basic plugin-management.
type BasePlugin interface {
	Start() error
	Stop() error
	Configure(map[string]interface{}) error
	Heartbeat() error
}

// basePluginRPCClient is the type that all pluginClients implicitly implement
// behind the scenes. We expose the direct types because we'd like to make sure
// that they are embeddedable (*BasePluginRPCClient and *BasePluginRPCServer)
// by plugin packages. However, we'd still like to be able to create a
// `BasePluginClient` type and we'd like it to _only_ use the interface. This
// allows us to pass in RPCClients from different plugin packages, of different
// types, as long as they meet this interface
type BasePluginClient interface {
	Start() error
	Stop() error
	Configure(map[string]interface{}) error
	Heartbeat() error
	Kill()
}

type basePluginRPCClient interface {
	Start() *gatekeeper.Error
	Stop() *gatekeeper.Error
	Configure(map[string]interface{}) *gatekeeper.Error
	Heartbeat() *gatekeeper.Error
}

func NewBasePluginClient(rpcClient basePluginRPCClient, client *plugin.Client) BasePluginClient {
	return &basePluginClient{
		rpcClient: rpcClient,
		client:    client,
	}
}

type basePluginClient struct {
	rpcClient basePluginRPCClient
	client    *plugin.Client
}

func (b *basePluginClient) Start() error {
	return b.rpcClient.Start()
}

func (b *basePluginClient) Stop() error {
	return b.rpcClient.Stop()
}

func (b *basePluginClient) Configure(args map[string]interface{}) error {
	return b.rpcClient.Configure(args)
}

func (b *basePluginClient) Heartbeat() error {
	return b.rpcClient.Heartbeat()
}

func (b *basePluginClient) Kill() {
	b.client.Kill()
}

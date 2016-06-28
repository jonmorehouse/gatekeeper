package event

import (
	"fmt"
	"net/rpc"
	"os/exec"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/shared"
)

var Handshake = plugin.HandshakeConfig{
	ProtocolVersion:  1,
	MagicCookieKey:   "gatekeeper|plugin-type",
	MagicCookieValue: "metric",
}

// Plugin exposes an interface that plugin creators can implement in order to
// receive Events in the plugin lifecycle. The RPC wiring around this abstracts
// away some of the type conversion and casting around errors and how we pass
// metrics around.
type Plugin interface {
	Configure(map[string]interface{}) error
	Heartbeat() error
	Start() error
	Stop() error

	EventMetric(*shared.EventMetric) error
	ProfilingMetric(*shared.ProfilingMetric) error
	PluginMetric(*shared.PluginMetric) error
	RequestMetric(*shared.RequestMetric) error
	UpstreamMetric(*shared.UpstreamMetric) error
}

// PluginClient exposes an interface that the gatekeeper process (or any user
// of the event.Plugin) interface above would interact with. Underneath the
// hood, it provides some abstraction to ensure that we are sending proper data
// types over the wire and that we interact with the PluginRPC interface correctly.
type PluginClient interface {
	Configure(map[string]interface{}) error
	Heartbeat() error
	Start() error
	Stop() error

	// kill the underlying RPC connection, this should not be required, but is handy
	Kill()

	// write metric in batches over RPC
	WriteEventMetrics([]*shared.EventMetric) []error
	WriteProfilingMetrics([]*shared.ProfilingMetric) []error
	WritePluginMetrics([]*shared.PluginMetric) []error
	WriteRequestMetrics([]*shared.RequestMetric) []error
	WriteUpstreamMetrics([]*shared.UpstreamMetric) []error
}

type pluginClient struct {
	// the underlying plugin connection that manages the plugin lifecycle
	// at the process level
	client *plugin.Client

	// interface that we expose over the wire for transferring metrics in batches
	pluginRPC PluginRPC
}

func NewPluginClient(client *plugin.Client, pluginRPC PluginRPC) PluginClient {
	return &pluginClient{
		client:    client,
		pluginRPC: pluginRPC,
	}
}

func (p *pluginClient) Configure(opts map[string]interface{}) error {
	return shared.ErrorToError(p.pluginRPC.Configure(opts))
}

func (p *pluginClient) Heartbeat() error {
	return shared.ErrorToError(p.pluginRPC.Heartbeat())
}

func (p *pluginClient) Start() error {
	return shared.ErrorToError(p.pluginRPC.Start())
}

func (p *pluginClient) Stop() error {
	return shared.ErrorToError(p.pluginRPC.Stop())
}

func (p *pluginClient) Kill() {
	p.client.Kill()
}

func sharedErrsToErrs(input []*shared.Error) []error {
	if len(input) == 0 {
		return nil
	}

	errs := make([]error, 0, len(input))
	for _, sharedErr := range input {
		err := shared.ErrorToError(sharedErr)
		if err != nil {
			errs = append(errs, err)
		}
	}

	return errs
}

func (p *pluginClient) WriteEventMetrics(metrics []*shared.EventMetric) []error {
	errs := p.pluginRPC.EventMetric(metrics)
	return sharedErrsToErrs(errs)
}

func (p *pluginClient) WriteProfilingMetrics(metrics []*shared.ProfilingMetric) []error {
	errs := p.pluginRPC.ProfilingMetric(metrics)
	return sharedErrsToErrs(errs)
}

func (p *pluginClient) WritePluginMetrics(metrics []*shared.PluginMetric) []error {
	errs := p.pluginRPC.PluginMetric(metrics)
	return sharedErrsToErrs(errs)
}

func (p *pluginClient) WriteRequestMetrics(metrics []*shared.RequestMetric) []error {
	errs := p.pluginRPC.RequestMetric(metrics)
	return sharedErrsToErrs(errs)
}

func (p *pluginClient) WriteUpstreamMetrics(metrics []*shared.UpstreamMetric) []error {
	errs := p.pluginRPC.UpstreamMetric(metrics)
	return sharedErrsToErrs(errs)
}

// PluginRPC is the type that is actually sent over the wire. Specifically, it
// exposes concrete types for all errors to be sent back and forth.
// NOTE, we send arrays of metrics over the wire, collecting errors and
// returning them. This is so we can reduce the number of round trips across
// the socket while also simplifying logic for the plugins implementing this
// interface.
type PluginRPC interface {
	Configure(map[string]interface{}) *shared.Error
	Heartbeat() *shared.Error
	Start() *shared.Error
	Stop() *shared.Error

	EventMetric([]*shared.EventMetric) []*shared.Error
	ProfilingMetric([]*shared.ProfilingMetric) []*shared.Error
	PluginMetric([]*shared.PluginMetric) []*shared.Error
	RequestMetric([]*shared.RequestMetric) []*shared.Error
	UpstreamMetric([]*shared.UpstreamMetric) []*shared.Error
}

// PluginDispenser is the type that go-plugin interacts with for handling
// Dispensary of client and server processes. Specifically it will _only_
// create RPCClient and RPCServer types for consistency's sake.
type PluginDispenser struct {
	EventPlugin Plugin
}

func (d PluginDispenser) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &RPCServer{broker: b, impl: d.EventPlugin}, nil
}

func (d PluginDispenser) Client(b *plugin.MuxBroker, c *rpc.Client) (interface{}, error) {
	return &RPCClient{broker: b, client: c}, nil
}

// RunPlugin should only be called by plugin processes. It accepts a type
// implementing the Plugin interface above and builds plumbing around it to
// call the various methods in the right context.
func RunPlugin(name string, eventPlugin Plugin) error {
	pluginDispenser := PluginDispenser{EventPlugin: eventPlugin}
	plugin.Serve(&plugin.ServeConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			name: &pluginDispenser,
		},
	})
	return nil
}

// NewClient creates a subprocess using the `go-plugin` provided plumbing and
// exposes an interface to interact with plugin in an intuitive way. Its worth
// mentioning that this `NewClient` is the only _blessed_ way to create an
// instance of the event plugin as it specifically wraps the raw RPCClient with
// a `PluginClient` type that allows for a nicer user interface.
func NewClient(name string, cmd string) (PluginClient, error) {
	pluginDispenser := PluginDispenser{}

	client := plugin.NewClient(&plugin.ClientConfig{
		HandshakeConfig: Handshake,
		Plugins: map[string]plugin.Plugin{
			name: &pluginDispenser,
		},
		Cmd: exec.Command(cmd),
	})

	rpcClient, err := client.Client()
	if err != nil {
		client.Kill()
		return nil, err
	}

	// fetch an instance of the RPCClient from over the wire
	rawPlugin, err := rpcClient.Dispense(name)
	if err != nil {
		client.Kill()
		return nil, err
	}

	// cast the rawPlugin into the PluginRPC type which exposes concrete errors over the wire
	pluginRPC, ok := rawPlugin.(PluginRPC)
	if !ok {
		client.Kill()
		return nil, fmt.Errorf("Unable to cast plugin to the correct type")
	}

	// finally, build a new PluginClient which exposes a friendly interface
	// for the gatekeeper process to use.
	return &pluginClient{
		// this is the go-plugin.Client
		client:    client,
		pluginRPC: pluginRPC,
	}, nil
}

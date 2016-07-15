package metric

import (
	"net/rpc"

	"github.com/hashicorp/go-plugin"
	"github.com/jonmorehouse/gatekeeper/internal"
)

type Dispenser struct {
	Impl Plugin
}

func (d *Dispenser) Server(b *plugin.MuxBroker) (interface{}, error) {
	return &RPCServer{
		d.Impl,
		b,
		internal.NewBasePluginRPCServer(b, d.Impl),
	}, nil
}

func (d *Dispenser) Client(b *plugin.MuxBroker, client *rpc.Client) (interface{}, error) {
	return &RPCClient{
		b,
		client,
		internal.NewBasePluginRPCClient(b, client),
	}, nil
}

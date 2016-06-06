package gatekeeper

import (
	"github.com/jonmorehouse/gatekeeper/shared"
)

type ResponseModifier interface {
	Modify(*shared.Response) error
}

type ResponseModifierClient interface {
	Modify(*shared.Response) error
}

type LocalResponseModifier struct{}

func (m *LocalResponseModifier) Modify(res *shared.Response) error {
	return nil
}

// TODO AsyncResponseModifier which leverages PluginManager to perform Response mods over RPC

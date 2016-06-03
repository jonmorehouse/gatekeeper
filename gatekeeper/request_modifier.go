package gatekeeper

import (
	"github.com/jonmorehouse/gatekeeper/shared"
)

type RequestModifier interface {
	Modify(*shared.Request) error
}

type LocalRequestModifier struct{}

func (m *LocalRequestModifier) Modify(req *shared.Request) error {
	return nil
}

// TODO AsyncRequestModifier which leverages PluginManager to perform Request mods over RPC

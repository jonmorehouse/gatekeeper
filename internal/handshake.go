package internal

import (
	"github.com/hashicorp/go-plugin"
)

func NewHandshakeConfig(pluginType string) plugin.HandshakeConfig {
	return plugin.HandshakeConfig{
		ProtocolVersion:  1,
		MagicCookieKey:   "gatekeeper|plugin-type",
		MagicCookieValue: pluginType,
	}
}

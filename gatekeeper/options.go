package gatekeeper

import (
	"errors"
	"os/exec"
	"strings"
	"time"
)

var UpstreamPluginRequiredError = errors.New("upstream-plugin required")
var MetricPluginRequiredError = errors.New("metric-plugin required")
var ModifierPluginRequiredError = errors.New("modifier-plugin required")
var LoadbalancerPluginRequiredError = errors.New("loadbalancer-plugin required")

var InvalidUpstreamPluginError = errors.New("invalid upstream-plugin")
var InvalidMetricPluginError = errors.New("invalid upstream-plugin")
var InvalidModifierPluginError = errors.New("invalid upstream-plugin")
var InvalidLoadbalancerPluginError = errors.New("invalid upstream-plugin")

var InvalidPluginArgs = errors.New("invalid plugin args")

var InvalidPluginTimeoutError = errors.New("invalid plugin-timeout")
var InvalidProxyTimeoutError = errors.New("invalid proxy-timeout")

type Options struct {
	// optional plugin configuration
	RouterPlugin        string
	RouterPluginArgs    map[string]interface{}
	RouterPluginEnabled bool

	LoadBalancerPlugin        string
	LoadBalancerPluginArgs    map[string]interface{}
	LoadBalancerPluginEnabled bool

	// required plugins configuration
	UpstreamPlugins    []string
	UpstreamPluginArgs map[string]interface{}

	ModifierPlugins    []string
	ModifierPluginArgs map[string]interface{}

	MetricPlugins    []string
	MetricPluginArgs map[string]interface{}

	// server configurations
	HTTPPublic     bool
	HTTPPublicPort uint

	HTTPInternal     bool
	HTTPInternalPort uint

	HTTPSPublic     bool
	HTTPSPublicPort uint

	HTTPSInternal     bool
	HTTPSInternalPort uint

	// Default timeout for upstream requests
	DefaultProxyTimeout time.Duration
	PluginTimeout       time.Duration
}

func ValidatePlugins(rawCmds []string) ([]string, error) {
	cmds := make([]string, len(rawCmds))

	for idx, cmd := range rawCmds {
		pieces := strings.SplitN(cmd, " ", 1)
		fullpath, err := exec.LookPath(pieces[0])
		if err != nil {
			return []string(nil), err
		}

		pieces[0] = fullpath
		cmds[idx] = strings.Join(pieces, " ")
	}

	return cmds, nil
}

func (o *Options) Validate() error {
	// Not implemented yet
	return nil
}

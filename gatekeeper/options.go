package gatekeeper

import (
	"fmt"
	"os/exec"
	"time"
)

type Options struct {
	// name of the plugin binary, expects a full path or the name of a
	// binary in PATH eg: `loadbalancer` or `/home/foo/bin/loadbalancer`
	UpstreamPlugins []string
	// number of instances to run
	UpstreamPluginsCount uint
	// Opts to be passed along to plugin. Not currently used
	UpstreamPluginOpts map[string]interface{}

	// name of the plugin binary, expects a full path or the name of a
	// binary in PATH eg: `loadbalancer` or `/home/foo/bin/loadbalancer`
	LoadBalancerPlugin string
	// number of instances to run of the loadBalancer
	LoadBalancerPluginsCount uint
	// Opts to be passed along to plugin. Not currently used
	LoadBalancerPluginOpts map[string]interface{}

	// name of the plugin binary, expects a full path or the name of a
	// binary in PATH eg: `modifier` or `/home/foo/bin/modifier`
	ModifierPlugins []string
	// number of instances to run
	ModifierPluginsCount uint
	// Opts to be passed along to plugin. Not currently used
	ModifierPluginOpts map[string]interface{}

	// name of the plugin binary, expects a full path or the name of a
	// binary in PATH eg: `event` or `/home/foo/bin/event`
	MetricPlugins []string
	// number of instances to run
	MetricPluginsCount uint
	// Opts to be passed along to plugin. Not currently used
	MetricPluginOpts map[string]interface{}

	// Ports to start servers listening on. If not provided, the server
	// will not be started. If collisions are detected, then this will
	// error out.
	HTTPPublicPort   uint
	HTTPInternalPort uint

	// Default timeout for upstream requests
	DefaultTimeout time.Duration
}

func ValidatePlugins(paths []string) ([]string, error) {
	errs := NewMultiError()
	validPaths := make([]string, 0, len(paths))

	for _, path := range paths {
		if fullpath, err := exec.LookPath(path); err != nil {
			errs.Add(err)
		} else {
			validPaths = append(validPaths, fullpath)
		}
	}

	return validPaths, errs.ToErr()
}

func (o *Options) Validate() error {
	errs := NewMultiError()

	// verify that Upstream plugins are configured correctly
	if plugins, err := ValidatePlugins(o.UpstreamPlugins); err != nil {
		errs.Add(err)
	} else {
		o.UpstreamPlugins = plugins
	}
	if o.UpstreamPluginsCount == 0 {
		return fmt.Errorf("UPSTREAM_PLUGIN_COUNT_ZERO")
	}

	if fullpath, err := exec.LookPath(o.LoadBalancerPlugin); err != nil {
		errs.Add(err)
	} else {
		o.LoadBalancerPlugin = fullpath
	}

	if o.LoadBalancerPluginsCount == 0 {
		return fmt.Errorf("LOAD_BALANCER_PLUGIN_COUNT_ZERO")
	}

	// TODO handle event and modifier plugins

	return errs.ToErr()
}

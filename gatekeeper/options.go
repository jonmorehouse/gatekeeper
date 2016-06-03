package main

import (
	"fmt"
	"os/exec"
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
	LoadbalancerPlugins []string
	// number of instances to run
	LoadbalancerPluginsCount uint
	// Opts to be passed along to plugin. Not currently used
	LoadBalancerPluginOpts map[string]interface{}

	// name of the plugin binary, expects a full path or the name of a
	// binary in PATH eg: `loadbalancer` or `/home/foo/bin/loadbalancer`
	RequestModifierPlugins []string
	// number of instances to run
	RequestModifierPluginsCount uint
	// Opts to be passed along to plugin. Not currently used
	RequestModifierPluginOpts map[string]interface{}

	// name of the plugin binary, expects a full path or the name of a
	// binary in PATH eg: `loadbalancer` or `/home/foo/bin/loadbalancer`
	ResponseModifierPlugins []string
	// number of instances to run
	ResponseModifierPluginsCount uint
	// Opts to be passed along to plugin. Not currently used
	ResponseModifierPluginOpts map[string]interface{}

	// Ports to start servers listening on. If not provided, the server
	// will not be started. If collisions are detected, then this will
	// error out.
	HttpPublicPort   uint
	HttpInternalPort uint
	TCPPublicPort    uint
	TCPInternalPort  uint
}

func ValidatePlugins(paths []string) ([]string, error) {
	errs := NewMultiError()
	validPaths := make([]string, 0, len(paths))

	for _, path := range paths {
		if fullpath, err := exec.LookPath(path); err != nil {
			errs.Add(err)
		} else {
			validPaths = append(fullpath, path)
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

	// verify that LoadBalancer plugins are configured correctly
	if plugins, err := ValidatePlugins(o.LoadBalancerPlugins); err != nil {
		errs.Add(err)
	} else {
		o.LoadBalancerPlugins = plugins
	}
	if o.LoadBalancerPluginsCount == 0 {
		return fmt.Errorf("LOAD_BALANCER_PLUGIN_COUNT_ZERO")
	}

	// verify that RequestModifier plugins are configured correctly
	if plugins, err := ValidatePlugins(o.LoadBalancerPlugins); err != nil {
		errs.Add(err)
	} else {
		o.LoadBalancerPlugins = plugins
	}

	// verify that ResponseModifier plugins are configured correctly
	if plugins, err := ValidatePlugins(o.LoadBalancerPlugins); err != nil {
		errs.Add(err)
	} else {
		o.LoadBalancerPlugins = plugins
	}

	return errs.ToErr()
}

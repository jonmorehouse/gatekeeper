package gatekeeper

import "errors"

// Global errors
var (
	UpstreamNotFoundErr = errors.New("upstream not found")
	BackendNotFoundErr  = errors.New("backend not found")
	RouteNotFoundErr    = errors.New("route now found")
)

// Plugin specific errors
var (
	NoManagerErr  = errors.New("no upstream_plugin.Manager available")
	NotStartedErr = errors.New("not started error")
	NoConfigErr   = errors.New("No config error")
)

package gatekeeper

import "errors"

// Global errors
var (
	UpstreamNotFoundErr = errors.New("upstream not found")
	BackendNotFoundErr  = errors.New("backend not found")
)

// UpstreamPlugin errors
var (
	NoManagerErr = errors.New("no upstream_plugin.Manager available")
)

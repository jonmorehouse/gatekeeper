package gatekeeper

import "errors"

var UpstreamNotFoundErr = errors.New("upstream not found")
var BackendNotFoundErr = errors.New("backend not found")

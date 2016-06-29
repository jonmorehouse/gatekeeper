package gatekeeper

import (
	"bytes"
	"sync"

	"github.com/jonmorehouse/gatekeeper/shared"
)

// internal-error represents a set of errors for capturing irregular behaviour
// inside of the gatekeeper core application.
type internalError uint

const (
	// internal errors that should not happen
	InternalError internalError = iota + 1
	InternalPluginError
	InternalTimeoutError
	InternalEventError
	InternalProxierError
	InternalBroadcastError

	// ConfigurationError
	ConfigurationError

	// Request Lifecycle Errors
	ResponseWriteError
	ServerShuttingDownError
	UpstreamNotFoundError
	BackendNotFoundError
	ProxyTimeoutError
	NoBackendsFoundError
	BackendAddressError

	// PluginErrors
	LoadBalancerPluginError
	ModifierPluginError
)

var internalErrorMapping = map[internalError]string{
	InternalError:          "internal error",
	InternalPluginError:    "internal plugin error",
	InternalTimeoutError:   "internal timeout error",
	InternalEventError:     "internal event error",
	InternalBroadcastError: "internal broadcast error",

	ConfigurationError: "invalid configuration",

	ServerShuttingDownError: "server shutting down",
	ResponseWriteError:      "response write error",
	UpstreamNotFoundError:   "upstream not found",
	BackendNotFoundError:    "backend not found",
	BackendAddressError:     "backend address error",
	NoBackendsFoundError:    "no upstream backends found",
	InternalProxierError:    "internal proxier error",
	LoadBalancerPluginError: "load balancer plugin error",
	ModifierPluginError:     "modifier plugin error",
	ProxyTimeoutError:       "proxy timeout error",
}

func (i internalError) String() string {
	desc, ok := internalErrorMapping[i]
	if !ok {
		shared.ProgrammingError("internalError string mapping not found")
	}
	return desc
}

func (i internalError) Error() string {
	return i.String()
}

// goroutine safe error implementing type for managing multiple errors
type MultiError struct {
	errors []error
	sync.RWMutex
}

func NewMultiError() *MultiError {
	return &MultiError{
		errors: make([]error, 0, 0),
	}
}

func (m *MultiError) Add(err error) {
	m.Lock()
	defer m.Unlock()
	m.errors = append(m.errors, err)
}

func (m *MultiError) Error() string {
	var buffer bytes.Buffer
	for _, e := range m.errors {
		buffer.WriteString(e.Error())
		buffer.WriteString("\n")
	}

	return buffer.String()
}

func (m *MultiError) ToErr() error {
	if len(m.errors) == 0 {
		return nil
	}

	return m
}

package core

import (
	"bytes"
	"errors"
)

var (
	InternalError = errors.New("internal error")

	// Plugin Specific Errors
	PluginTimeoutErr           = errors.New("plugin timeout error")
	InternalPluginError        = errors.New("internal plugin error")
	InternalTimeoutError       = errors.New("internal timeout error")
	InternalBroadcastError     = errors.New("internal broadcast error")
	InternalEventListenerError = errors.New("internal event listener error")
	InvalidEventError          = errors.New("invalid event error")
	UnsubscribedEventError     = errors.New("unsubscribed event error")
	RouteNotFoundError         = errors.New("No route found error")
	InvalidUpstreamEventErr    = errors.New("invalid upstream event")

	// Configuration error
	ConfigurationError = errors.New("invalid configuration")

	// Specific errors
	ServerShuttingDownError = errors.New("server shutting down")
	ResponseWriteError      = errors.New("response write error")

	UpstreamNotFoundError    = errors.New("upstream not found")
	UpstreamDuplicateIDError = errors.New("duplicate upstream ID error")

	BackendDuplicateIDError = errors.New("duplicate backend ID error")
	BackendNotFoundError    = errors.New("backend not found")
	BackendAddressError     = errors.New("invalid backend address error")
	NoBackendsFoundError    = errors.New("no upstream backends found")
	OrphanedBackendError    = errors.New("orphaned backend error")

	InternalProxierError    = errors.New("internal proxier error")
	LoadBalancerPluginError = errors.New("load balancer plugin error")
	ModifierPluginError     = errors.New("modifier plugin error")
	ProxyTimeoutError       = errors.New("proxy timeout error")

	InvalidEventErr      = errors.New("invalid event error")
	InvalidPluginErr     = errors.New("invalid plugin type error")
	DuplicateUpstreamErr = errors.New("duplicate upstream error")
	DuplicateBackendErr  = errors.New("duplicate backend error")
	BackendAddressErr    = errors.New("invalid backend error")
)

// goroutine safe error implementing type for managing multiple errors
type MultiError struct {
	errs []error
	RWMutex
}

func NewMultiError() *MultiError {
	return &MultiError{
		errs: make([]error, 0, 0),
	}
}

func (m *MultiError) Add(err error) {
	if err == nil || reflect.IsNil(err) {
		return
	}

	m.Lock()
	defer m.Unlock()
	m.errs = append(m.errs, err)
}

func (m *MultiError) Error() string {
	var buffer bytes.Buffer
	for _, e := range m.errs {
		buffer.WriteString(e.Error())
		buffer.WriteString("\n")
	}

	return buffer.String()
}

func (m *MultiError) ToErr() error {
	if len(m.errs) == 0 {
		return nil
	}

	return m
}

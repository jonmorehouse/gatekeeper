package core

import (
	"bytes"
	"errors"
	"sync"
)

var InternalError = errors.New("internal error")

var PluginTimeoutError = errors.New("plugin timeout error")
var InternalPluginError = errors.New("internal plugin error")
var InternalTimeoutError = errors.New("internal timeout error")
var InternalBroadcastError = errors.New("internal broadcast error")
var InternalEventListenerError = errors.New("internal event listener error")
var InvalidEventError = errors.New("invalid event error")
var UnsubscribedEventError = errors.New("unsubscribed event error")
var RouteNotFoundError = errors.New("No route found error")
var InvalidUpstreamEventErr = errors.New("invalid upstream event")

var ConfigurationError = errors.New("invalid configuration")

var ServerShuttingDownError = errors.New("server shutting down")
var ResponseWriteError = errors.New("response write error")

var UpstreamNotFoundError = errors.New("upstream not found")
var UpstreamDuplicateIDError = errors.New("duplicate upstream ID error")

var BackendDuplicateIDError = errors.New("duplicate backend ID error")
var BackendNotFoundError = errors.New("backend not found")
var BackendAddressError = errors.New("invalid backend address error")
var NoBackendsFoundError = errors.New("no upstream backends found")
var OrphanedBackendError = errors.New("orphaned backend error")

var InternalProxierError = errors.New("internal proxier error")
var LoadBalancerPluginError = errors.New("load balancer plugin error")
var ModifierPluginError = errors.New("modifier plugin error")
var ProxyTimeoutError = errors.New("proxy timeout error")

var InvalidEventErr = errors.New("invalid event error")
var DuplicateUpstreamErr = errors.New("duplicate upstream error")
var DuplicateBackendErr = errors.New("duplicate backend error")
var BackendAddressErr = errors.New("invalid backend error")

// goroutine safe error implementing type for managing multiple errors
type MultiError struct {
	errs []error
	sync.RWMutex
}

func NewMultiError() *MultiError {
	return &MultiError{
		errs: make([]error, 0, 0),
	}
}

func (m *MultiError) Add(err error) {
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

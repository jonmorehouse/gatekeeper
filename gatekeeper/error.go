package gatekeeper

import (
	"bytes"
	"sync"
)

type MultiError struct {
	errors []error
}

func NewMultiError() *MultiError {
	return &MultiError{}
}

func (m *MultiError) Add(err error) {
	if m.errors == nil {
		m.errors = make([]error, 0, 1)
	}
	m.errors = append(m.errors, err)
}

func (m MultiError) Error() string {
	var buffer bytes.Buffer
	for _, e := range m.errors {
		buffer.WriteString(e.Error())
		buffer.WriteString("\n")
	}

	return buffer.String()
}

func (m MultiError) ToErr() error {
	if len(m.errors) == 0 {
		return nil
	}

	return m
}

// goroutine safe error type
type AsyncMultiError struct {
	errors []error
	sync.RWMutex
}

func NewAsyncMultiError() *AsyncMultiError {
	return &AsyncMultiError{}
}

func (m *AsyncMultiError) Add(err error) {
	m.Lock()
	defer m.Unlock()
	m.errors = append(m.errors, err)
}

func (m *AsyncMultiError) Error() string {
	var buffer bytes.Buffer
	for _, e := range m.errors {
		buffer.WriteString(e.Error())
		buffer.WriteString("\n")
	}

	return buffer.String()
}

func (m *AsyncMultiError) ToErr() error {
	if len(m.errors) == 0 {
		return nil
	}

	return m
}

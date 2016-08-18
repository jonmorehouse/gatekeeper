package test

import (
	"fmt"
	"reflect"
	"testing"
)

type Mock interface {
	SetCallSideEffect(method string, cb func(...interface{}))
	SetCallResp(method string, resp ...interface{})

	GetCalls(method string) [][]interface{}

	AssertCalledWith(method string, args ...interface{})
	AssertCalled(method string)
	AssertCallCount(method string, count int)
}

func NewMock(t *testing.T) Mock {
	return &mock{
		t:     t,
		calls: make(map[string][][]interface{}),
	}
}

type mock struct {
	t *testing.T

	calls       map[string][][]interface{}
	resp        map[string][]interface{}
	sideEffects map[string]func(...interface{})
}

func (m *mock) SetCallSideEffect(method string, cb func(...interface{})) {
	m.sideEffects[method] = cb
}

func (m *mock) SetCallResp(method string, resp ...interface{}) {
	m.resp[method] = resp
}

func (m *mock) GetCalls(method string) [][]interface{} {
	calls, ok := m.calls[method]
	AssertTrue(m.t, ok)
	return calls
}

func (m *mock) AssertCalledWith(method string, args ...interface{}) {
	calls, ok := m.calls[method]
	AssertTrue(m.t, ok)
	AssertTrue(m.t, len(calls) > 0)

	// iterate through each call
	for _, call := range calls {
		if len(call) != len(args) {
			continue
		}

		// for each call, walk through each argument and check it
		// against the input args
		equal := true
		for idx, arg := range call {
			if !reflect.DeepEqual(arg, args[idx]) {
				equal = false
				break
			}
		}

		// if the entirety of the call matched the correct args return
		// because the test passes
		if equal {
			return
		}
	}

	fail(m.t, fmt.Sprintf("did not find call with args %v", args))
}

func (m *mock) AssertNotCalled(method string) {
	calls, ok := m.calls[method]
	AssertTrue(m.t, (ok && len(calls) == 0) || !ok)
}

func (m *mock) AssertCalled(method string) {
	calls, ok := m.calls[method]
	AssertTrue(m.t, ok)
	AssertTrue(m.t, len(calls) > 0)
}

func (m *mock) AssertCallCount(method string, count int) {
	calls, ok := m.calls[method]

	AssertTrue(m.t, ok)
	AssertEqual(m.t, count, len(calls))
}

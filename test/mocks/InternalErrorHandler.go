// Code generated by mockery v2.52.1. DO NOT EDIT.

package mocks

import mock "github.com/stretchr/testify/mock"

// InternalErrorHandler is an autogenerated mock type for the InternalErrorHandler type
type InternalErrorHandler struct {
	mock.Mock
}

// HandleError provides a mock function with given fields: err
func (_m *InternalErrorHandler) HandleError(err error) {
	_m.Called(err)
}

// NewInternalErrorHandler creates a new instance of InternalErrorHandler. It also registers a testing interface on the mock and a cleanup function to assert the mocks expectations.
// The first argument is typically a *testing.T value.
func NewInternalErrorHandler(t interface {
	mock.TestingT
	Cleanup(func())
}) *InternalErrorHandler {
	mock := &InternalErrorHandler{}
	mock.Mock.Test(t)

	t.Cleanup(func() { mock.AssertExpectations(t) })

	return mock
}

// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package subscribehandler -destination ./mock_subscribehandler_subscribehandler.go -source main.go -build_flags=-mod=mod
//

// Package subscribehandler is a generated GoMock package.
package subscribehandler

import (
	reflect "reflect"

	gomock "go.uber.org/mock/gomock"
)

// MockSubscribeHandler is a mock of SubscribeHandler interface.
type MockSubscribeHandler struct {
	ctrl     *gomock.Controller
	recorder *MockSubscribeHandlerMockRecorder
	isgomock struct{}
}

// MockSubscribeHandlerMockRecorder is the mock recorder for MockSubscribeHandler.
type MockSubscribeHandlerMockRecorder struct {
	mock *MockSubscribeHandler
}

// NewMockSubscribeHandler creates a new mock instance.
func NewMockSubscribeHandler(ctrl *gomock.Controller) *MockSubscribeHandler {
	mock := &MockSubscribeHandler{ctrl: ctrl}
	mock.recorder = &MockSubscribeHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockSubscribeHandler) EXPECT() *MockSubscribeHandlerMockRecorder {
	return m.recorder
}

// Run mocks base method.
func (m *MockSubscribeHandler) Run() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Run")
	ret0, _ := ret[0].(error)
	return ret0
}

// Run indicates an expected call of Run.
func (mr *MockSubscribeHandlerMockRecorder) Run() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Run", reflect.TypeOf((*MockSubscribeHandler)(nil).Run))
}

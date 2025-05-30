// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package websockhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod
//

// Package websockhandler is a generated GoMock package.
package websockhandler

import (
	context "context"
	agent "monorepo/bin-agent-manager/models/agent"
	externalmedia "monorepo/bin-call-manager/models/externalmedia"
	http "net/http"
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// MockWebsockHandler is a mock of WebsockHandler interface.
type MockWebsockHandler struct {
	ctrl     *gomock.Controller
	recorder *MockWebsockHandlerMockRecorder
	isgomock struct{}
}

// MockWebsockHandlerMockRecorder is the mock recorder for MockWebsockHandler.
type MockWebsockHandlerMockRecorder struct {
	mock *MockWebsockHandler
}

// NewMockWebsockHandler creates a new mock instance.
func NewMockWebsockHandler(ctrl *gomock.Controller) *MockWebsockHandler {
	mock := &MockWebsockHandler{ctrl: ctrl}
	mock.recorder = &MockWebsockHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockWebsockHandler) EXPECT() *MockWebsockHandlerMockRecorder {
	return m.recorder
}

// RunMediaStream mocks base method.
func (m *MockWebsockHandler) RunMediaStream(ctx context.Context, w http.ResponseWriter, r *http.Request, referenceType externalmedia.ReferenceType, referenceID uuid.UUID, encapsulation string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RunMediaStream", ctx, w, r, referenceType, referenceID, encapsulation)
	ret0, _ := ret[0].(error)
	return ret0
}

// RunMediaStream indicates an expected call of RunMediaStream.
func (mr *MockWebsockHandlerMockRecorder) RunMediaStream(ctx, w, r, referenceType, referenceID, encapsulation any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RunMediaStream", reflect.TypeOf((*MockWebsockHandler)(nil).RunMediaStream), ctx, w, r, referenceType, referenceID, encapsulation)
}

// RunSubscription mocks base method.
func (m *MockWebsockHandler) RunSubscription(ctx context.Context, w http.ResponseWriter, r *http.Request, a *agent.Agent) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RunSubscription", ctx, w, r, a)
	ret0, _ := ret[0].(error)
	return ret0
}

// RunSubscription indicates an expected call of RunSubscription.
func (mr *MockWebsockHandlerMockRecorder) RunSubscription(ctx, w, r, a any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RunSubscription", reflect.TypeOf((*MockWebsockHandler)(nil).RunSubscription), ctx, w, r, a)
}

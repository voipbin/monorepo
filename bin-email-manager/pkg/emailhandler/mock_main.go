// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package emailhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod
//

// Package emailhandler is a generated GoMock package.
package emailhandler

import (
	context "context"
	address "monorepo/bin-common-handler/models/address"
	email "monorepo/bin-email-manager/models/email"
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// MockEmailHandler is a mock of EmailHandler interface.
type MockEmailHandler struct {
	ctrl     *gomock.Controller
	recorder *MockEmailHandlerMockRecorder
	isgomock struct{}
}

// MockEmailHandlerMockRecorder is the mock recorder for MockEmailHandler.
type MockEmailHandlerMockRecorder struct {
	mock *MockEmailHandler
}

// NewMockEmailHandler creates a new mock instance.
func NewMockEmailHandler(ctrl *gomock.Controller) *MockEmailHandler {
	mock := &MockEmailHandler{ctrl: ctrl}
	mock.recorder = &MockEmailHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockEmailHandler) EXPECT() *MockEmailHandlerMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockEmailHandler) Create(ctx context.Context, customerID, activeflowID uuid.UUID, destinations []address.Address, subject, content string, attachments []email.Attachment) (*email.Email, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, customerID, activeflowID, destinations, subject, content, attachments)
	ret0, _ := ret[0].(*email.Email)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockEmailHandlerMockRecorder) Create(ctx, customerID, activeflowID, destinations, subject, content, attachments any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockEmailHandler)(nil).Create), ctx, customerID, activeflowID, destinations, subject, content, attachments)
}

// Delete mocks base method.
func (m *MockEmailHandler) Delete(ctx context.Context, id uuid.UUID) (*email.Email, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(*email.Email)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete.
func (mr *MockEmailHandlerMockRecorder) Delete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockEmailHandler)(nil).Delete), ctx, id)
}

// Get mocks base method.
func (m *MockEmailHandler) Get(ctx context.Context, id uuid.UUID) (*email.Email, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, id)
	ret0, _ := ret[0].(*email.Email)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockEmailHandlerMockRecorder) Get(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockEmailHandler)(nil).Get), ctx, id)
}

// Gets mocks base method.
func (m *MockEmailHandler) Gets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*email.Email, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Gets", ctx, token, size, filters)
	ret0, _ := ret[0].([]*email.Email)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Gets indicates an expected call of Gets.
func (mr *MockEmailHandlerMockRecorder) Gets(ctx, token, size, filters any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Gets", reflect.TypeOf((*MockEmailHandler)(nil).Gets), ctx, token, size, filters)
}

// Hook mocks base method.
func (m *MockEmailHandler) Hook(ctx context.Context, uri string, data []byte) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Hook", ctx, uri, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// Hook indicates an expected call of Hook.
func (mr *MockEmailHandlerMockRecorder) Hook(ctx, uri, data any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Hook", reflect.TypeOf((*MockEmailHandler)(nil).Hook), ctx, uri, data)
}

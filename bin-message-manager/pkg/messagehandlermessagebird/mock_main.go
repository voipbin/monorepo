// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package messagehandlermessagebird -destination ./mock_main.go -source main.go -build_flags=-mod=mod
//

// Package messagehandlermessagebird is a generated GoMock package.
package messagehandlermessagebird

import (
	context "context"
	address "monorepo/bin-common-handler/models/address"
	target "monorepo/bin-message-manager/models/target"
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// MockMessageHandlerMessagebird is a mock of MessageHandlerMessagebird interface.
type MockMessageHandlerMessagebird struct {
	ctrl     *gomock.Controller
	recorder *MockMessageHandlerMessagebirdMockRecorder
	isgomock struct{}
}

// MockMessageHandlerMessagebirdMockRecorder is the mock recorder for MockMessageHandlerMessagebird.
type MockMessageHandlerMessagebirdMockRecorder struct {
	mock *MockMessageHandlerMessagebird
}

// NewMockMessageHandlerMessagebird creates a new mock instance.
func NewMockMessageHandlerMessagebird(ctrl *gomock.Controller) *MockMessageHandlerMessagebird {
	mock := &MockMessageHandlerMessagebird{ctrl: ctrl}
	mock.recorder = &MockMessageHandlerMessagebirdMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMessageHandlerMessagebird) EXPECT() *MockMessageHandlerMessagebirdMockRecorder {
	return m.recorder
}

// SendMessage mocks base method.
func (m *MockMessageHandlerMessagebird) SendMessage(ctx context.Context, messageID uuid.UUID, source *address.Address, targets []target.Target, text string) ([]target.Target, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SendMessage", ctx, messageID, source, targets, text)
	ret0, _ := ret[0].([]target.Target)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// SendMessage indicates an expected call of SendMessage.
func (mr *MockMessageHandlerMessagebirdMockRecorder) SendMessage(ctx, messageID, source, targets, text any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SendMessage", reflect.TypeOf((*MockMessageHandlerMessagebird)(nil).SendMessage), ctx, messageID, source, targets, text)
}

// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package variablehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod
//

// Package variablehandler is a generated GoMock package.
package variablehandler

import (
	context "context"
	variable "monorepo/bin-flow-manager/models/variable"
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// MockVariableHandler is a mock of VariableHandler interface.
type MockVariableHandler struct {
	ctrl     *gomock.Controller
	recorder *MockVariableHandlerMockRecorder
	isgomock struct{}
}

// MockVariableHandlerMockRecorder is the mock recorder for MockVariableHandler.
type MockVariableHandlerMockRecorder struct {
	mock *MockVariableHandler
}

// NewMockVariableHandler creates a new mock instance.
func NewMockVariableHandler(ctrl *gomock.Controller) *MockVariableHandler {
	mock := &MockVariableHandler{ctrl: ctrl}
	mock.recorder = &MockVariableHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockVariableHandler) EXPECT() *MockVariableHandlerMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockVariableHandler) Create(ctx context.Context, activeflowID uuid.UUID, variables map[string]string) (*variable.Variable, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, activeflowID, variables)
	ret0, _ := ret[0].(*variable.Variable)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockVariableHandlerMockRecorder) Create(ctx, activeflowID, variables any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockVariableHandler)(nil).Create), ctx, activeflowID, variables)
}

// DeleteVariable mocks base method.
func (m *MockVariableHandler) DeleteVariable(ctx context.Context, id uuid.UUID, key string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "DeleteVariable", ctx, id, key)
	ret0, _ := ret[0].(error)
	return ret0
}

// DeleteVariable indicates an expected call of DeleteVariable.
func (mr *MockVariableHandlerMockRecorder) DeleteVariable(ctx, id, key any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "DeleteVariable", reflect.TypeOf((*MockVariableHandler)(nil).DeleteVariable), ctx, id, key)
}

// Get mocks base method.
func (m *MockVariableHandler) Get(ctx context.Context, id uuid.UUID) (*variable.Variable, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, id)
	ret0, _ := ret[0].(*variable.Variable)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockVariableHandlerMockRecorder) Get(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockVariableHandler)(nil).Get), ctx, id)
}

// Set mocks base method.
func (m *MockVariableHandler) Set(ctx context.Context, t *variable.Variable) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Set", ctx, t)
	ret0, _ := ret[0].(error)
	return ret0
}

// Set indicates an expected call of Set.
func (mr *MockVariableHandlerMockRecorder) Set(ctx, t any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Set", reflect.TypeOf((*MockVariableHandler)(nil).Set), ctx, t)
}

// SetVariable mocks base method.
func (m *MockVariableHandler) SetVariable(ctx context.Context, id uuid.UUID, variables map[string]string) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SetVariable", ctx, id, variables)
	ret0, _ := ret[0].(error)
	return ret0
}

// SetVariable indicates an expected call of SetVariable.
func (mr *MockVariableHandlerMockRecorder) SetVariable(ctx, id, variables any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SetVariable", reflect.TypeOf((*MockVariableHandler)(nil).SetVariable), ctx, id, variables)
}

// Substitute mocks base method.
func (m *MockVariableHandler) Substitute(ctx context.Context, id uuid.UUID, data string) (string, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Substitute", ctx, id, data)
	ret0, _ := ret[0].(string)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Substitute indicates an expected call of Substitute.
func (mr *MockVariableHandlerMockRecorder) Substitute(ctx, id, data any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Substitute", reflect.TypeOf((*MockVariableHandler)(nil).Substitute), ctx, id, data)
}

// SubstituteByte mocks base method.
func (m *MockVariableHandler) SubstituteByte(ctx context.Context, data []byte, v *variable.Variable) []byte {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubstituteByte", ctx, data, v)
	ret0, _ := ret[0].([]byte)
	return ret0
}

// SubstituteByte indicates an expected call of SubstituteByte.
func (mr *MockVariableHandlerMockRecorder) SubstituteByte(ctx, data, v any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubstituteByte", reflect.TypeOf((*MockVariableHandler)(nil).SubstituteByte), ctx, data, v)
}

// SubstituteOption mocks base method.
func (m *MockVariableHandler) SubstituteOption(ctx context.Context, data map[string]any, vars *variable.Variable) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "SubstituteOption", ctx, data, vars)
}

// SubstituteOption indicates an expected call of SubstituteOption.
func (mr *MockVariableHandlerMockRecorder) SubstituteOption(ctx, data, vars any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubstituteOption", reflect.TypeOf((*MockVariableHandler)(nil).SubstituteOption), ctx, data, vars)
}

// SubstituteString mocks base method.
func (m *MockVariableHandler) SubstituteString(ctx context.Context, data string, v *variable.Variable) string {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "SubstituteString", ctx, data, v)
	ret0, _ := ret[0].(string)
	return ret0
}

// SubstituteString indicates an expected call of SubstituteString.
func (mr *MockVariableHandlerMockRecorder) SubstituteString(ctx, data, v any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "SubstituteString", reflect.TypeOf((*MockVariableHandler)(nil).SubstituteString), ctx, data, v)
}

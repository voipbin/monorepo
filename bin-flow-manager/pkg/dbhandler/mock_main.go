// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod
//

// Package dbhandler is a generated GoMock package.
package dbhandler

import (
	context "context"
	activeflow "monorepo/bin-flow-manager/models/activeflow"
	flow "monorepo/bin-flow-manager/models/flow"
	variable "monorepo/bin-flow-manager/models/variable"
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// MockDBHandler is a mock of DBHandler interface.
type MockDBHandler struct {
	ctrl     *gomock.Controller
	recorder *MockDBHandlerMockRecorder
	isgomock struct{}
}

// MockDBHandlerMockRecorder is the mock recorder for MockDBHandler.
type MockDBHandlerMockRecorder struct {
	mock *MockDBHandler
}

// NewMockDBHandler creates a new mock instance.
func NewMockDBHandler(ctrl *gomock.Controller) *MockDBHandler {
	mock := &MockDBHandler{ctrl: ctrl}
	mock.recorder = &MockDBHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockDBHandler) EXPECT() *MockDBHandlerMockRecorder {
	return m.recorder
}

// ActiveflowCreate mocks base method.
func (m *MockDBHandler) ActiveflowCreate(ctx context.Context, af *activeflow.Activeflow) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActiveflowCreate", ctx, af)
	ret0, _ := ret[0].(error)
	return ret0
}

// ActiveflowCreate indicates an expected call of ActiveflowCreate.
func (mr *MockDBHandlerMockRecorder) ActiveflowCreate(ctx, af any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActiveflowCreate", reflect.TypeOf((*MockDBHandler)(nil).ActiveflowCreate), ctx, af)
}

// ActiveflowDelete mocks base method.
func (m *MockDBHandler) ActiveflowDelete(ctx context.Context, id uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActiveflowDelete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// ActiveflowDelete indicates an expected call of ActiveflowDelete.
func (mr *MockDBHandlerMockRecorder) ActiveflowDelete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActiveflowDelete", reflect.TypeOf((*MockDBHandler)(nil).ActiveflowDelete), ctx, id)
}

// ActiveflowGet mocks base method.
func (m *MockDBHandler) ActiveflowGet(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActiveflowGet", ctx, id)
	ret0, _ := ret[0].(*activeflow.Activeflow)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ActiveflowGet indicates an expected call of ActiveflowGet.
func (mr *MockDBHandlerMockRecorder) ActiveflowGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActiveflowGet", reflect.TypeOf((*MockDBHandler)(nil).ActiveflowGet), ctx, id)
}

// ActiveflowGetWithLock mocks base method.
func (m *MockDBHandler) ActiveflowGetWithLock(ctx context.Context, id uuid.UUID) (*activeflow.Activeflow, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActiveflowGetWithLock", ctx, id)
	ret0, _ := ret[0].(*activeflow.Activeflow)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ActiveflowGetWithLock indicates an expected call of ActiveflowGetWithLock.
func (mr *MockDBHandlerMockRecorder) ActiveflowGetWithLock(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActiveflowGetWithLock", reflect.TypeOf((*MockDBHandler)(nil).ActiveflowGetWithLock), ctx, id)
}

// ActiveflowGets mocks base method.
func (m *MockDBHandler) ActiveflowGets(ctx context.Context, token string, size uint64, filters map[activeflow.Field]any) ([]*activeflow.Activeflow, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActiveflowGets", ctx, token, size, filters)
	ret0, _ := ret[0].([]*activeflow.Activeflow)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ActiveflowGets indicates an expected call of ActiveflowGets.
func (mr *MockDBHandlerMockRecorder) ActiveflowGets(ctx, token, size, filters any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActiveflowGets", reflect.TypeOf((*MockDBHandler)(nil).ActiveflowGets), ctx, token, size, filters)
}

// ActiveflowReleaseLock mocks base method.
func (m *MockDBHandler) ActiveflowReleaseLock(ctx context.Context, id uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActiveflowReleaseLock", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// ActiveflowReleaseLock indicates an expected call of ActiveflowReleaseLock.
func (mr *MockDBHandlerMockRecorder) ActiveflowReleaseLock(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActiveflowReleaseLock", reflect.TypeOf((*MockDBHandler)(nil).ActiveflowReleaseLock), ctx, id)
}

// ActiveflowUpdate mocks base method.
func (m *MockDBHandler) ActiveflowUpdate(ctx context.Context, id uuid.UUID, fields map[activeflow.Field]any) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ActiveflowUpdate", ctx, id, fields)
	ret0, _ := ret[0].(error)
	return ret0
}

// ActiveflowUpdate indicates an expected call of ActiveflowUpdate.
func (mr *MockDBHandlerMockRecorder) ActiveflowUpdate(ctx, id, fields any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ActiveflowUpdate", reflect.TypeOf((*MockDBHandler)(nil).ActiveflowUpdate), ctx, id, fields)
}

// FlowCreate mocks base method.
func (m *MockDBHandler) FlowCreate(ctx context.Context, f *flow.Flow) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FlowCreate", ctx, f)
	ret0, _ := ret[0].(error)
	return ret0
}

// FlowCreate indicates an expected call of FlowCreate.
func (mr *MockDBHandlerMockRecorder) FlowCreate(ctx, f any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FlowCreate", reflect.TypeOf((*MockDBHandler)(nil).FlowCreate), ctx, f)
}

// FlowDelete mocks base method.
func (m *MockDBHandler) FlowDelete(ctx context.Context, id uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FlowDelete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// FlowDelete indicates an expected call of FlowDelete.
func (mr *MockDBHandlerMockRecorder) FlowDelete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FlowDelete", reflect.TypeOf((*MockDBHandler)(nil).FlowDelete), ctx, id)
}

// FlowGet mocks base method.
func (m *MockDBHandler) FlowGet(ctx context.Context, id uuid.UUID) (*flow.Flow, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FlowGet", ctx, id)
	ret0, _ := ret[0].(*flow.Flow)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FlowGet indicates an expected call of FlowGet.
func (mr *MockDBHandlerMockRecorder) FlowGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FlowGet", reflect.TypeOf((*MockDBHandler)(nil).FlowGet), ctx, id)
}

// FlowGets mocks base method.
func (m *MockDBHandler) FlowGets(ctx context.Context, token string, size uint64, filters map[flow.Field]any) ([]*flow.Flow, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FlowGets", ctx, token, size, filters)
	ret0, _ := ret[0].([]*flow.Flow)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FlowGets indicates an expected call of FlowGets.
func (mr *MockDBHandlerMockRecorder) FlowGets(ctx, token, size, filters any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FlowGets", reflect.TypeOf((*MockDBHandler)(nil).FlowGets), ctx, token, size, filters)
}

// FlowSetToCache mocks base method.
func (m *MockDBHandler) FlowSetToCache(ctx context.Context, f *flow.Flow) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FlowSetToCache", ctx, f)
	ret0, _ := ret[0].(error)
	return ret0
}

// FlowSetToCache indicates an expected call of FlowSetToCache.
func (mr *MockDBHandlerMockRecorder) FlowSetToCache(ctx, f any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FlowSetToCache", reflect.TypeOf((*MockDBHandler)(nil).FlowSetToCache), ctx, f)
}

// FlowUpdate mocks base method.
func (m *MockDBHandler) FlowUpdate(ctx context.Context, id uuid.UUID, fields map[flow.Field]any) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FlowUpdate", ctx, id, fields)
	ret0, _ := ret[0].(error)
	return ret0
}

// FlowUpdate indicates an expected call of FlowUpdate.
func (mr *MockDBHandlerMockRecorder) FlowUpdate(ctx, id, fields any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FlowUpdate", reflect.TypeOf((*MockDBHandler)(nil).FlowUpdate), ctx, id, fields)
}

// VariableCreate mocks base method.
func (m *MockDBHandler) VariableCreate(ctx context.Context, t *variable.Variable) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VariableCreate", ctx, t)
	ret0, _ := ret[0].(error)
	return ret0
}

// VariableCreate indicates an expected call of VariableCreate.
func (mr *MockDBHandlerMockRecorder) VariableCreate(ctx, t any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VariableCreate", reflect.TypeOf((*MockDBHandler)(nil).VariableCreate), ctx, t)
}

// VariableGet mocks base method.
func (m *MockDBHandler) VariableGet(ctx context.Context, id uuid.UUID) (*variable.Variable, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VariableGet", ctx, id)
	ret0, _ := ret[0].(*variable.Variable)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// VariableGet indicates an expected call of VariableGet.
func (mr *MockDBHandlerMockRecorder) VariableGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VariableGet", reflect.TypeOf((*MockDBHandler)(nil).VariableGet), ctx, id)
}

// VariableUpdate mocks base method.
func (m *MockDBHandler) VariableUpdate(ctx context.Context, t *variable.Variable) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "VariableUpdate", ctx, t)
	ret0, _ := ret[0].(error)
	return ret0
}

// VariableUpdate indicates an expected call of VariableUpdate.
func (mr *MockDBHandlerMockRecorder) VariableUpdate(ctx, t any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "VariableUpdate", reflect.TypeOf((*MockDBHandler)(nil).VariableUpdate), ctx, t)
}

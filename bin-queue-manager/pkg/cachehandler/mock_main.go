// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package cachehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod
//

// Package cachehandler is a generated GoMock package.
package cachehandler

import (
	context "context"
	queue "monorepo/bin-queue-manager/models/queue"
	queuecall "monorepo/bin-queue-manager/models/queuecall"
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// MockCacheHandler is a mock of CacheHandler interface.
type MockCacheHandler struct {
	ctrl     *gomock.Controller
	recorder *MockCacheHandlerMockRecorder
	isgomock struct{}
}

// MockCacheHandlerMockRecorder is the mock recorder for MockCacheHandler.
type MockCacheHandlerMockRecorder struct {
	mock *MockCacheHandler
}

// NewMockCacheHandler creates a new mock instance.
func NewMockCacheHandler(ctrl *gomock.Controller) *MockCacheHandler {
	mock := &MockCacheHandler{ctrl: ctrl}
	mock.recorder = &MockCacheHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockCacheHandler) EXPECT() *MockCacheHandlerMockRecorder {
	return m.recorder
}

// Connect mocks base method.
func (m *MockCacheHandler) Connect() error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Connect")
	ret0, _ := ret[0].(error)
	return ret0
}

// Connect indicates an expected call of Connect.
func (mr *MockCacheHandlerMockRecorder) Connect() *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Connect", reflect.TypeOf((*MockCacheHandler)(nil).Connect))
}

// QueueGet mocks base method.
func (m *MockCacheHandler) QueueGet(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueueGet", ctx, id)
	ret0, _ := ret[0].(*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueueGet indicates an expected call of QueueGet.
func (mr *MockCacheHandlerMockRecorder) QueueGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueueGet", reflect.TypeOf((*MockCacheHandler)(nil).QueueGet), ctx, id)
}

// QueueSet mocks base method.
func (m *MockCacheHandler) QueueSet(ctx context.Context, u *queue.Queue) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueueSet", ctx, u)
	ret0, _ := ret[0].(error)
	return ret0
}

// QueueSet indicates an expected call of QueueSet.
func (mr *MockCacheHandlerMockRecorder) QueueSet(ctx, u any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueueSet", reflect.TypeOf((*MockCacheHandler)(nil).QueueSet), ctx, u)
}

// QueuecallGet mocks base method.
func (m *MockCacheHandler) QueuecallGet(ctx context.Context, id uuid.UUID) (*queuecall.Queuecall, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueuecallGet", ctx, id)
	ret0, _ := ret[0].(*queuecall.Queuecall)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueuecallGet indicates an expected call of QueuecallGet.
func (mr *MockCacheHandlerMockRecorder) QueuecallGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueuecallGet", reflect.TypeOf((*MockCacheHandler)(nil).QueuecallGet), ctx, id)
}

// QueuecallGetByReferenceID mocks base method.
func (m *MockCacheHandler) QueuecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*queuecall.Queuecall, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueuecallGetByReferenceID", ctx, referenceID)
	ret0, _ := ret[0].(*queuecall.Queuecall)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// QueuecallGetByReferenceID indicates an expected call of QueuecallGetByReferenceID.
func (mr *MockCacheHandlerMockRecorder) QueuecallGetByReferenceID(ctx, referenceID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueuecallGetByReferenceID", reflect.TypeOf((*MockCacheHandler)(nil).QueuecallGetByReferenceID), ctx, referenceID)
}

// QueuecallSet mocks base method.
func (m *MockCacheHandler) QueuecallSet(ctx context.Context, u *queuecall.Queuecall) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "QueuecallSet", ctx, u)
	ret0, _ := ret[0].(error)
	return ret0
}

// QueuecallSet indicates an expected call of QueuecallSet.
func (mr *MockCacheHandlerMockRecorder) QueuecallSet(ctx, u any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "QueuecallSet", reflect.TypeOf((*MockCacheHandler)(nil).QueuecallSet), ctx, u)
}

// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package cachehandler -destination ./mock_cachehandler.go -source main.go -build_flags=-mod=mod
//

// Package cachehandler is a generated GoMock package.
package cachehandler

import (
	context "context"
	streaming "monorepo/bin-transcribe-manager/models/streaming"
	transcribe "monorepo/bin-transcribe-manager/models/transcribe"
	transcript "monorepo/bin-transcribe-manager/models/transcript"
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

// StreamingGet mocks base method.
func (m *MockCacheHandler) StreamingGet(ctx context.Context, id uuid.UUID) (*streaming.Streaming, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StreamingGet", ctx, id)
	ret0, _ := ret[0].(*streaming.Streaming)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// StreamingGet indicates an expected call of StreamingGet.
func (mr *MockCacheHandlerMockRecorder) StreamingGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StreamingGet", reflect.TypeOf((*MockCacheHandler)(nil).StreamingGet), ctx, id)
}

// StreamingSet mocks base method.
func (m *MockCacheHandler) StreamingSet(ctx context.Context, stream *streaming.Streaming) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "StreamingSet", ctx, stream)
	ret0, _ := ret[0].(error)
	return ret0
}

// StreamingSet indicates an expected call of StreamingSet.
func (mr *MockCacheHandlerMockRecorder) StreamingSet(ctx, stream any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "StreamingSet", reflect.TypeOf((*MockCacheHandler)(nil).StreamingSet), ctx, stream)
}

// TranscribeGet mocks base method.
func (m *MockCacheHandler) TranscribeGet(ctx context.Context, id uuid.UUID) (*transcribe.Transcribe, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TranscribeGet", ctx, id)
	ret0, _ := ret[0].(*transcribe.Transcribe)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TranscribeGet indicates an expected call of TranscribeGet.
func (mr *MockCacheHandlerMockRecorder) TranscribeGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TranscribeGet", reflect.TypeOf((*MockCacheHandler)(nil).TranscribeGet), ctx, id)
}

// TranscribeSet mocks base method.
func (m *MockCacheHandler) TranscribeSet(ctx context.Context, trans *transcribe.Transcribe) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TranscribeSet", ctx, trans)
	ret0, _ := ret[0].(error)
	return ret0
}

// TranscribeSet indicates an expected call of TranscribeSet.
func (mr *MockCacheHandlerMockRecorder) TranscribeSet(ctx, trans any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TranscribeSet", reflect.TypeOf((*MockCacheHandler)(nil).TranscribeSet), ctx, trans)
}

// TranscriptGet mocks base method.
func (m *MockCacheHandler) TranscriptGet(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TranscriptGet", ctx, id)
	ret0, _ := ret[0].(*transcript.Transcript)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// TranscriptGet indicates an expected call of TranscriptGet.
func (mr *MockCacheHandlerMockRecorder) TranscriptGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TranscriptGet", reflect.TypeOf((*MockCacheHandler)(nil).TranscriptGet), ctx, id)
}

// TranscriptSet mocks base method.
func (m *MockCacheHandler) TranscriptSet(ctx context.Context, trans *transcript.Transcript) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "TranscriptSet", ctx, trans)
	ret0, _ := ret[0].(error)
	return ret0
}

// TranscriptSet indicates an expected call of TranscriptSet.
func (mr *MockCacheHandlerMockRecorder) TranscriptSet(ctx, trans any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "TranscriptSet", reflect.TypeOf((*MockCacheHandler)(nil).TranscriptSet), ctx, trans)
}

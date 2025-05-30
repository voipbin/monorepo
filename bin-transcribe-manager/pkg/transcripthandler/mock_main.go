// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package transcripthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod
//

// Package transcripthandler is a generated GoMock package.
package transcripthandler

import (
	context "context"
	transcript "monorepo/bin-transcribe-manager/models/transcript"
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// MockTranscriptHandler is a mock of TranscriptHandler interface.
type MockTranscriptHandler struct {
	ctrl     *gomock.Controller
	recorder *MockTranscriptHandlerMockRecorder
	isgomock struct{}
}

// MockTranscriptHandlerMockRecorder is the mock recorder for MockTranscriptHandler.
type MockTranscriptHandlerMockRecorder struct {
	mock *MockTranscriptHandler
}

// NewMockTranscriptHandler creates a new mock instance.
func NewMockTranscriptHandler(ctrl *gomock.Controller) *MockTranscriptHandler {
	mock := &MockTranscriptHandler{ctrl: ctrl}
	mock.recorder = &MockTranscriptHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockTranscriptHandler) EXPECT() *MockTranscriptHandlerMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockTranscriptHandler) Create(ctx context.Context, customerID, transcribeID uuid.UUID, direction transcript.Direction, message, tmTranscript string) (*transcript.Transcript, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, customerID, transcribeID, direction, message, tmTranscript)
	ret0, _ := ret[0].(*transcript.Transcript)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockTranscriptHandlerMockRecorder) Create(ctx, customerID, transcribeID, direction, message, tmTranscript any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockTranscriptHandler)(nil).Create), ctx, customerID, transcribeID, direction, message, tmTranscript)
}

// Delete mocks base method.
func (m *MockTranscriptHandler) Delete(ctx context.Context, id uuid.UUID) (*transcript.Transcript, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(*transcript.Transcript)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete.
func (mr *MockTranscriptHandlerMockRecorder) Delete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockTranscriptHandler)(nil).Delete), ctx, id)
}

// Gets mocks base method.
func (m *MockTranscriptHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*transcript.Transcript, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Gets", ctx, size, token, filters)
	ret0, _ := ret[0].([]*transcript.Transcript)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Gets indicates an expected call of Gets.
func (mr *MockTranscriptHandlerMockRecorder) Gets(ctx, size, token, filters any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Gets", reflect.TypeOf((*MockTranscriptHandler)(nil).Gets), ctx, size, token, filters)
}

// Recording mocks base method.
func (m *MockTranscriptHandler) Recording(ctx context.Context, customerID, transcribeID, recordingID uuid.UUID, language string) ([]*transcript.Transcript, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Recording", ctx, customerID, transcribeID, recordingID, language)
	ret0, _ := ret[0].([]*transcript.Transcript)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Recording indicates an expected call of Recording.
func (mr *MockTranscriptHandlerMockRecorder) Recording(ctx, customerID, transcribeID, recordingID, language any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Recording", reflect.TypeOf((*MockTranscriptHandler)(nil).Recording), ctx, customerID, transcribeID, recordingID, language)
}

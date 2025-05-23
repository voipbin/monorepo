// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package dbhandler -destination ./mock_dbhandler_dbhandler.go -source main.go -build_flags=-mod=mod
//

// Package dbhandler is a generated GoMock package.
package dbhandler

import (
	context "context"
	conference "monorepo/bin-conference-manager/models/conference"
	conferencecall "monorepo/bin-conference-manager/models/conferencecall"
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

// ConferenceAddConferencecallID mocks base method.
func (m *MockDBHandler) ConferenceAddConferencecallID(ctx context.Context, id, callID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceAddConferencecallID", ctx, id, callID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferenceAddConferencecallID indicates an expected call of ConferenceAddConferencecallID.
func (mr *MockDBHandlerMockRecorder) ConferenceAddConferencecallID(ctx, id, callID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceAddConferencecallID", reflect.TypeOf((*MockDBHandler)(nil).ConferenceAddConferencecallID), ctx, id, callID)
}

// ConferenceAddRecordingIDs mocks base method.
func (m *MockDBHandler) ConferenceAddRecordingIDs(ctx context.Context, id, recordingID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceAddRecordingIDs", ctx, id, recordingID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferenceAddRecordingIDs indicates an expected call of ConferenceAddRecordingIDs.
func (mr *MockDBHandlerMockRecorder) ConferenceAddRecordingIDs(ctx, id, recordingID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceAddRecordingIDs", reflect.TypeOf((*MockDBHandler)(nil).ConferenceAddRecordingIDs), ctx, id, recordingID)
}

// ConferenceAddTranscribeIDs mocks base method.
func (m *MockDBHandler) ConferenceAddTranscribeIDs(ctx context.Context, id, transcribeID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceAddTranscribeIDs", ctx, id, transcribeID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferenceAddTranscribeIDs indicates an expected call of ConferenceAddTranscribeIDs.
func (mr *MockDBHandlerMockRecorder) ConferenceAddTranscribeIDs(ctx, id, transcribeID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceAddTranscribeIDs", reflect.TypeOf((*MockDBHandler)(nil).ConferenceAddTranscribeIDs), ctx, id, transcribeID)
}

// ConferenceCreate mocks base method.
func (m *MockDBHandler) ConferenceCreate(ctx context.Context, cf *conference.Conference) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceCreate", ctx, cf)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferenceCreate indicates an expected call of ConferenceCreate.
func (mr *MockDBHandlerMockRecorder) ConferenceCreate(ctx, cf any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceCreate", reflect.TypeOf((*MockDBHandler)(nil).ConferenceCreate), ctx, cf)
}

// ConferenceDelete mocks base method.
func (m *MockDBHandler) ConferenceDelete(ctx context.Context, id uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceDelete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferenceDelete indicates an expected call of ConferenceDelete.
func (mr *MockDBHandlerMockRecorder) ConferenceDelete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceDelete", reflect.TypeOf((*MockDBHandler)(nil).ConferenceDelete), ctx, id)
}

// ConferenceEnd mocks base method.
func (m *MockDBHandler) ConferenceEnd(ctx context.Context, id uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceEnd", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferenceEnd indicates an expected call of ConferenceEnd.
func (mr *MockDBHandlerMockRecorder) ConferenceEnd(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceEnd", reflect.TypeOf((*MockDBHandler)(nil).ConferenceEnd), ctx, id)
}

// ConferenceGet mocks base method.
func (m *MockDBHandler) ConferenceGet(ctx context.Context, id uuid.UUID) (*conference.Conference, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceGet", ctx, id)
	ret0, _ := ret[0].(*conference.Conference)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConferenceGet indicates an expected call of ConferenceGet.
func (mr *MockDBHandlerMockRecorder) ConferenceGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceGet", reflect.TypeOf((*MockDBHandler)(nil).ConferenceGet), ctx, id)
}

// ConferenceGetByConfbridgeID mocks base method.
func (m *MockDBHandler) ConferenceGetByConfbridgeID(ctx context.Context, confbridgeID uuid.UUID) (*conference.Conference, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceGetByConfbridgeID", ctx, confbridgeID)
	ret0, _ := ret[0].(*conference.Conference)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConferenceGetByConfbridgeID indicates an expected call of ConferenceGetByConfbridgeID.
func (mr *MockDBHandlerMockRecorder) ConferenceGetByConfbridgeID(ctx, confbridgeID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceGetByConfbridgeID", reflect.TypeOf((*MockDBHandler)(nil).ConferenceGetByConfbridgeID), ctx, confbridgeID)
}

// ConferenceGets mocks base method.
func (m *MockDBHandler) ConferenceGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*conference.Conference, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceGets", ctx, size, token, filters)
	ret0, _ := ret[0].([]*conference.Conference)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConferenceGets indicates an expected call of ConferenceGets.
func (mr *MockDBHandlerMockRecorder) ConferenceGets(ctx, size, token, filters any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceGets", reflect.TypeOf((*MockDBHandler)(nil).ConferenceGets), ctx, size, token, filters)
}

// ConferenceRemoveConferencecallID mocks base method.
func (m *MockDBHandler) ConferenceRemoveConferencecallID(ctx context.Context, id, callID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceRemoveConferencecallID", ctx, id, callID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferenceRemoveConferencecallID indicates an expected call of ConferenceRemoveConferencecallID.
func (mr *MockDBHandlerMockRecorder) ConferenceRemoveConferencecallID(ctx, id, callID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceRemoveConferencecallID", reflect.TypeOf((*MockDBHandler)(nil).ConferenceRemoveConferencecallID), ctx, id, callID)
}

// ConferenceSet mocks base method.
func (m *MockDBHandler) ConferenceSet(ctx context.Context, id uuid.UUID, name, detail string, data map[string]any, timeout int, preFlowID, postFlowID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceSet", ctx, id, name, detail, data, timeout, preFlowID, postFlowID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferenceSet indicates an expected call of ConferenceSet.
func (mr *MockDBHandlerMockRecorder) ConferenceSet(ctx, id, name, detail, data, timeout, preFlowID, postFlowID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceSet", reflect.TypeOf((*MockDBHandler)(nil).ConferenceSet), ctx, id, name, detail, data, timeout, preFlowID, postFlowID)
}

// ConferenceSetData mocks base method.
func (m *MockDBHandler) ConferenceSetData(ctx context.Context, id uuid.UUID, data map[string]any) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceSetData", ctx, id, data)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferenceSetData indicates an expected call of ConferenceSetData.
func (mr *MockDBHandlerMockRecorder) ConferenceSetData(ctx, id, data any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceSetData", reflect.TypeOf((*MockDBHandler)(nil).ConferenceSetData), ctx, id, data)
}

// ConferenceSetRecordingID mocks base method.
func (m *MockDBHandler) ConferenceSetRecordingID(ctx context.Context, id, recordingID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceSetRecordingID", ctx, id, recordingID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferenceSetRecordingID indicates an expected call of ConferenceSetRecordingID.
func (mr *MockDBHandlerMockRecorder) ConferenceSetRecordingID(ctx, id, recordingID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceSetRecordingID", reflect.TypeOf((*MockDBHandler)(nil).ConferenceSetRecordingID), ctx, id, recordingID)
}

// ConferenceSetStatus mocks base method.
func (m *MockDBHandler) ConferenceSetStatus(ctx context.Context, id uuid.UUID, status conference.Status) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceSetStatus", ctx, id, status)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferenceSetStatus indicates an expected call of ConferenceSetStatus.
func (mr *MockDBHandlerMockRecorder) ConferenceSetStatus(ctx, id, status any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceSetStatus", reflect.TypeOf((*MockDBHandler)(nil).ConferenceSetStatus), ctx, id, status)
}

// ConferenceSetTranscribeID mocks base method.
func (m *MockDBHandler) ConferenceSetTranscribeID(ctx context.Context, id, transcribeID uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferenceSetTranscribeID", ctx, id, transcribeID)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferenceSetTranscribeID indicates an expected call of ConferenceSetTranscribeID.
func (mr *MockDBHandlerMockRecorder) ConferenceSetTranscribeID(ctx, id, transcribeID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferenceSetTranscribeID", reflect.TypeOf((*MockDBHandler)(nil).ConferenceSetTranscribeID), ctx, id, transcribeID)
}

// ConferencecallCreate mocks base method.
func (m *MockDBHandler) ConferencecallCreate(ctx context.Context, cf *conferencecall.Conferencecall) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferencecallCreate", ctx, cf)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferencecallCreate indicates an expected call of ConferencecallCreate.
func (mr *MockDBHandlerMockRecorder) ConferencecallCreate(ctx, cf any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferencecallCreate", reflect.TypeOf((*MockDBHandler)(nil).ConferencecallCreate), ctx, cf)
}

// ConferencecallDelete mocks base method.
func (m *MockDBHandler) ConferencecallDelete(ctx context.Context, id uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferencecallDelete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferencecallDelete indicates an expected call of ConferencecallDelete.
func (mr *MockDBHandlerMockRecorder) ConferencecallDelete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferencecallDelete", reflect.TypeOf((*MockDBHandler)(nil).ConferencecallDelete), ctx, id)
}

// ConferencecallGet mocks base method.
func (m *MockDBHandler) ConferencecallGet(ctx context.Context, id uuid.UUID) (*conferencecall.Conferencecall, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferencecallGet", ctx, id)
	ret0, _ := ret[0].(*conferencecall.Conferencecall)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConferencecallGet indicates an expected call of ConferencecallGet.
func (mr *MockDBHandlerMockRecorder) ConferencecallGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferencecallGet", reflect.TypeOf((*MockDBHandler)(nil).ConferencecallGet), ctx, id)
}

// ConferencecallGetByReferenceID mocks base method.
func (m *MockDBHandler) ConferencecallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*conferencecall.Conferencecall, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferencecallGetByReferenceID", ctx, referenceID)
	ret0, _ := ret[0].(*conferencecall.Conferencecall)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConferencecallGetByReferenceID indicates an expected call of ConferencecallGetByReferenceID.
func (mr *MockDBHandlerMockRecorder) ConferencecallGetByReferenceID(ctx, referenceID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferencecallGetByReferenceID", reflect.TypeOf((*MockDBHandler)(nil).ConferencecallGetByReferenceID), ctx, referenceID)
}

// ConferencecallGets mocks base method.
func (m *MockDBHandler) ConferencecallGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*conferencecall.Conferencecall, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferencecallGets", ctx, size, token, filters)
	ret0, _ := ret[0].([]*conferencecall.Conferencecall)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// ConferencecallGets indicates an expected call of ConferencecallGets.
func (mr *MockDBHandlerMockRecorder) ConferencecallGets(ctx, size, token, filters any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferencecallGets", reflect.TypeOf((*MockDBHandler)(nil).ConferencecallGets), ctx, size, token, filters)
}

// ConferencecallUpdateStatus mocks base method.
func (m *MockDBHandler) ConferencecallUpdateStatus(ctx context.Context, id uuid.UUID, status conferencecall.Status) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "ConferencecallUpdateStatus", ctx, id, status)
	ret0, _ := ret[0].(error)
	return ret0
}

// ConferencecallUpdateStatus indicates an expected call of ConferencecallUpdateStatus.
func (mr *MockDBHandlerMockRecorder) ConferencecallUpdateStatus(ctx, id, status any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "ConferencecallUpdateStatus", reflect.TypeOf((*MockDBHandler)(nil).ConferencecallUpdateStatus), ctx, id, status)
}

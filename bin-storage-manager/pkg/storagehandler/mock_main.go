// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package storagehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod
//

// Package storagehandler is a generated GoMock package.
package storagehandler

import (
	context "context"
	bucketfile "monorepo/bin-storage-manager/models/bucketfile"
	compress_file "monorepo/bin-storage-manager/models/compressfile"
	file "monorepo/bin-storage-manager/models/file"
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// MockStorageHandler is a mock of StorageHandler interface.
type MockStorageHandler struct {
	ctrl     *gomock.Controller
	recorder *MockStorageHandlerMockRecorder
	isgomock struct{}
}

// MockStorageHandlerMockRecorder is the mock recorder for MockStorageHandler.
type MockStorageHandlerMockRecorder struct {
	mock *MockStorageHandler
}

// NewMockStorageHandler creates a new mock instance.
func NewMockStorageHandler(ctrl *gomock.Controller) *MockStorageHandler {
	mock := &MockStorageHandler{ctrl: ctrl}
	mock.recorder = &MockStorageHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockStorageHandler) EXPECT() *MockStorageHandlerMockRecorder {
	return m.recorder
}

// CompressfileCreate mocks base method.
func (m *MockStorageHandler) CompressfileCreate(ctx context.Context, referenceIDs, fileIDs []uuid.UUID) (*compress_file.CompressFile, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "CompressfileCreate", ctx, referenceIDs, fileIDs)
	ret0, _ := ret[0].(*compress_file.CompressFile)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// CompressfileCreate indicates an expected call of CompressfileCreate.
func (mr *MockStorageHandlerMockRecorder) CompressfileCreate(ctx, referenceIDs, fileIDs any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "CompressfileCreate", reflect.TypeOf((*MockStorageHandler)(nil).CompressfileCreate), ctx, referenceIDs, fileIDs)
}

// FileCreate mocks base method.
func (m *MockStorageHandler) FileCreate(ctx context.Context, customerID, ownerID uuid.UUID, referenceType file.ReferenceType, referenceID uuid.UUID, name, detail, filename, bucketName, filepath string) (*file.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FileCreate", ctx, customerID, ownerID, referenceType, referenceID, name, detail, filename, bucketName, filepath)
	ret0, _ := ret[0].(*file.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FileCreate indicates an expected call of FileCreate.
func (mr *MockStorageHandlerMockRecorder) FileCreate(ctx, customerID, ownerID, referenceType, referenceID, name, detail, filename, bucketName, filepath any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FileCreate", reflect.TypeOf((*MockStorageHandler)(nil).FileCreate), ctx, customerID, ownerID, referenceType, referenceID, name, detail, filename, bucketName, filepath)
}

// FileDelete mocks base method.
func (m *MockStorageHandler) FileDelete(ctx context.Context, id uuid.UUID) (*file.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FileDelete", ctx, id)
	ret0, _ := ret[0].(*file.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FileDelete indicates an expected call of FileDelete.
func (mr *MockStorageHandlerMockRecorder) FileDelete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FileDelete", reflect.TypeOf((*MockStorageHandler)(nil).FileDelete), ctx, id)
}

// FileGet mocks base method.
func (m *MockStorageHandler) FileGet(ctx context.Context, id uuid.UUID) (*file.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FileGet", ctx, id)
	ret0, _ := ret[0].(*file.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FileGet indicates an expected call of FileGet.
func (mr *MockStorageHandlerMockRecorder) FileGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FileGet", reflect.TypeOf((*MockStorageHandler)(nil).FileGet), ctx, id)
}

// FileGets mocks base method.
func (m *MockStorageHandler) FileGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*file.File, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "FileGets", ctx, token, size, filters)
	ret0, _ := ret[0].([]*file.File)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// FileGets indicates an expected call of FileGets.
func (mr *MockStorageHandlerMockRecorder) FileGets(ctx, token, size, filters any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "FileGets", reflect.TypeOf((*MockStorageHandler)(nil).FileGets), ctx, token, size, filters)
}

// RecordingDelete mocks base method.
func (m *MockStorageHandler) RecordingDelete(ctx context.Context, id uuid.UUID) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecordingDelete", ctx, id)
	ret0, _ := ret[0].(error)
	return ret0
}

// RecordingDelete indicates an expected call of RecordingDelete.
func (mr *MockStorageHandlerMockRecorder) RecordingDelete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecordingDelete", reflect.TypeOf((*MockStorageHandler)(nil).RecordingDelete), ctx, id)
}

// RecordingGet mocks base method.
func (m *MockStorageHandler) RecordingGet(ctx context.Context, id uuid.UUID) (*bucketfile.BucketFile, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RecordingGet", ctx, id)
	ret0, _ := ret[0].(*bucketfile.BucketFile)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RecordingGet indicates an expected call of RecordingGet.
func (mr *MockStorageHandlerMockRecorder) RecordingGet(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RecordingGet", reflect.TypeOf((*MockStorageHandler)(nil).RecordingGet), ctx, id)
}

// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package messagechatroomhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod
//

// Package messagechatroomhandler is a generated GoMock package.
package messagechatroomhandler

import (
	context "context"
	media "monorepo/bin-chat-manager/models/media"
	messagechatroom "monorepo/bin-chat-manager/models/messagechatroom"
	address "monorepo/bin-common-handler/models/address"
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// MockMessagechatroomHandler is a mock of MessagechatroomHandler interface.
type MockMessagechatroomHandler struct {
	ctrl     *gomock.Controller
	recorder *MockMessagechatroomHandlerMockRecorder
	isgomock struct{}
}

// MockMessagechatroomHandlerMockRecorder is the mock recorder for MockMessagechatroomHandler.
type MockMessagechatroomHandlerMockRecorder struct {
	mock *MockMessagechatroomHandler
}

// NewMockMessagechatroomHandler creates a new mock instance.
func NewMockMessagechatroomHandler(ctrl *gomock.Controller) *MockMessagechatroomHandler {
	mock := &MockMessagechatroomHandler{ctrl: ctrl}
	mock.recorder = &MockMessagechatroomHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockMessagechatroomHandler) EXPECT() *MockMessagechatroomHandlerMockRecorder {
	return m.recorder
}

// Create mocks base method.
func (m *MockMessagechatroomHandler) Create(ctx context.Context, customerID, ownerID, chatroomID, messagechatID uuid.UUID, source *address.Address, messageType messagechatroom.Type, text string, medias []media.Media) (*messagechatroom.Messagechatroom, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, customerID, ownerID, chatroomID, messagechatID, source, messageType, text, medias)
	ret0, _ := ret[0].(*messagechatroom.Messagechatroom)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockMessagechatroomHandlerMockRecorder) Create(ctx, customerID, ownerID, chatroomID, messagechatID, source, messageType, text, medias any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockMessagechatroomHandler)(nil).Create), ctx, customerID, ownerID, chatroomID, messagechatID, source, messageType, text, medias)
}

// Delete mocks base method.
func (m *MockMessagechatroomHandler) Delete(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(*messagechatroom.Messagechatroom)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete.
func (mr *MockMessagechatroomHandlerMockRecorder) Delete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockMessagechatroomHandler)(nil).Delete), ctx, id)
}

// Get mocks base method.
func (m *MockMessagechatroomHandler) Get(ctx context.Context, id uuid.UUID) (*messagechatroom.Messagechatroom, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, id)
	ret0, _ := ret[0].(*messagechatroom.Messagechatroom)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockMessagechatroomHandlerMockRecorder) Get(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockMessagechatroomHandler)(nil).Get), ctx, id)
}

// Gets mocks base method.
func (m *MockMessagechatroomHandler) Gets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*messagechatroom.Messagechatroom, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Gets", ctx, token, size, filters)
	ret0, _ := ret[0].([]*messagechatroom.Messagechatroom)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Gets indicates an expected call of Gets.
func (mr *MockMessagechatroomHandlerMockRecorder) Gets(ctx, token, size, filters any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Gets", reflect.TypeOf((*MockMessagechatroomHandler)(nil).Gets), ctx, token, size, filters)
}

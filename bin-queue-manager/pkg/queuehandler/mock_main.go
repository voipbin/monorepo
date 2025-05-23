// Code generated by MockGen. DO NOT EDIT.
// Source: main.go
//
// Generated by this command:
//
//	mockgen -package queuehandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod
//

// Package queuehandler is a generated GoMock package.
package queuehandler

import (
	context "context"
	agent "monorepo/bin-agent-manager/models/agent"
	customer "monorepo/bin-customer-manager/models/customer"
	queue "monorepo/bin-queue-manager/models/queue"
	reflect "reflect"

	uuid "github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// MockQueueHandler is a mock of QueueHandler interface.
type MockQueueHandler struct {
	ctrl     *gomock.Controller
	recorder *MockQueueHandlerMockRecorder
	isgomock struct{}
}

// MockQueueHandlerMockRecorder is the mock recorder for MockQueueHandler.
type MockQueueHandlerMockRecorder struct {
	mock *MockQueueHandler
}

// NewMockQueueHandler creates a new mock instance.
func NewMockQueueHandler(ctrl *gomock.Controller) *MockQueueHandler {
	mock := &MockQueueHandler{ctrl: ctrl}
	mock.recorder = &MockQueueHandlerMockRecorder{mock}
	return mock
}

// EXPECT returns an object that allows the caller to indicate expected use.
func (m *MockQueueHandler) EXPECT() *MockQueueHandlerMockRecorder {
	return m.recorder
}

// AddServiceQueuecallID mocks base method.
func (m *MockQueueHandler) AddServiceQueuecallID(ctx context.Context, id, queuecallID uuid.UUID) (*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddServiceQueuecallID", ctx, id, queuecallID)
	ret0, _ := ret[0].(*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddServiceQueuecallID indicates an expected call of AddServiceQueuecallID.
func (mr *MockQueueHandlerMockRecorder) AddServiceQueuecallID(ctx, id, queuecallID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddServiceQueuecallID", reflect.TypeOf((*MockQueueHandler)(nil).AddServiceQueuecallID), ctx, id, queuecallID)
}

// AddWaitQueueCallID mocks base method.
func (m *MockQueueHandler) AddWaitQueueCallID(ctx context.Context, id, queuecallID uuid.UUID) (*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "AddWaitQueueCallID", ctx, id, queuecallID)
	ret0, _ := ret[0].(*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// AddWaitQueueCallID indicates an expected call of AddWaitQueueCallID.
func (mr *MockQueueHandlerMockRecorder) AddWaitQueueCallID(ctx, id, queuecallID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "AddWaitQueueCallID", reflect.TypeOf((*MockQueueHandler)(nil).AddWaitQueueCallID), ctx, id, queuecallID)
}

// Create mocks base method.
func (m *MockQueueHandler) Create(ctx context.Context, customerID uuid.UUID, name, detail string, routingMethod queue.RoutingMethod, tagIDs []uuid.UUID, waitFlowID uuid.UUID, waitTimeout, serviceTimeout int) (*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Create", ctx, customerID, name, detail, routingMethod, tagIDs, waitFlowID, waitTimeout, serviceTimeout)
	ret0, _ := ret[0].(*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Create indicates an expected call of Create.
func (mr *MockQueueHandlerMockRecorder) Create(ctx, customerID, name, detail, routingMethod, tagIDs, waitFlowID, waitTimeout, serviceTimeout any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Create", reflect.TypeOf((*MockQueueHandler)(nil).Create), ctx, customerID, name, detail, routingMethod, tagIDs, waitFlowID, waitTimeout, serviceTimeout)
}

// Delete mocks base method.
func (m *MockQueueHandler) Delete(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Delete", ctx, id)
	ret0, _ := ret[0].(*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Delete indicates an expected call of Delete.
func (mr *MockQueueHandlerMockRecorder) Delete(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Delete", reflect.TypeOf((*MockQueueHandler)(nil).Delete), ctx, id)
}

// EventCUCustomerDeleted mocks base method.
func (m *MockQueueHandler) EventCUCustomerDeleted(ctx context.Context, cu *customer.Customer) error {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "EventCUCustomerDeleted", ctx, cu)
	ret0, _ := ret[0].(error)
	return ret0
}

// EventCUCustomerDeleted indicates an expected call of EventCUCustomerDeleted.
func (mr *MockQueueHandlerMockRecorder) EventCUCustomerDeleted(ctx, cu any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "EventCUCustomerDeleted", reflect.TypeOf((*MockQueueHandler)(nil).EventCUCustomerDeleted), ctx, cu)
}

// Execute mocks base method.
func (m *MockQueueHandler) Execute(ctx context.Context, id uuid.UUID) {
	m.ctrl.T.Helper()
	m.ctrl.Call(m, "Execute", ctx, id)
}

// Execute indicates an expected call of Execute.
func (mr *MockQueueHandlerMockRecorder) Execute(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Execute", reflect.TypeOf((*MockQueueHandler)(nil).Execute), ctx, id)
}

// Get mocks base method.
func (m *MockQueueHandler) Get(ctx context.Context, id uuid.UUID) (*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Get", ctx, id)
	ret0, _ := ret[0].(*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Get indicates an expected call of Get.
func (mr *MockQueueHandlerMockRecorder) Get(ctx, id any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Get", reflect.TypeOf((*MockQueueHandler)(nil).Get), ctx, id)
}

// GetAgents mocks base method.
func (m *MockQueueHandler) GetAgents(ctx context.Context, id uuid.UUID, status agent.Status) ([]agent.Agent, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "GetAgents", ctx, id, status)
	ret0, _ := ret[0].([]agent.Agent)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// GetAgents indicates an expected call of GetAgents.
func (mr *MockQueueHandlerMockRecorder) GetAgents(ctx, id, status any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "GetAgents", reflect.TypeOf((*MockQueueHandler)(nil).GetAgents), ctx, id, status)
}

// Gets mocks base method.
func (m *MockQueueHandler) Gets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "Gets", ctx, size, token, filters)
	ret0, _ := ret[0].([]*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// Gets indicates an expected call of Gets.
func (mr *MockQueueHandlerMockRecorder) Gets(ctx, size, token, filters any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "Gets", reflect.TypeOf((*MockQueueHandler)(nil).Gets), ctx, size, token, filters)
}

// RemoveQueuecallID mocks base method.
func (m *MockQueueHandler) RemoveQueuecallID(ctx context.Context, id, queuecallID uuid.UUID) (*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveQueuecallID", ctx, id, queuecallID)
	ret0, _ := ret[0].(*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RemoveQueuecallID indicates an expected call of RemoveQueuecallID.
func (mr *MockQueueHandlerMockRecorder) RemoveQueuecallID(ctx, id, queuecallID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveQueuecallID", reflect.TypeOf((*MockQueueHandler)(nil).RemoveQueuecallID), ctx, id, queuecallID)
}

// RemoveServiceQueuecallID mocks base method.
func (m *MockQueueHandler) RemoveServiceQueuecallID(ctx context.Context, id, queuecallID uuid.UUID) (*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "RemoveServiceQueuecallID", ctx, id, queuecallID)
	ret0, _ := ret[0].(*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// RemoveServiceQueuecallID indicates an expected call of RemoveServiceQueuecallID.
func (mr *MockQueueHandlerMockRecorder) RemoveServiceQueuecallID(ctx, id, queuecallID any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "RemoveServiceQueuecallID", reflect.TypeOf((*MockQueueHandler)(nil).RemoveServiceQueuecallID), ctx, id, queuecallID)
}

// UpdateBasicInfo mocks base method.
func (m *MockQueueHandler) UpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string, routingMethod queue.RoutingMethod, tagIDs []uuid.UUID, waitFlowID uuid.UUID, waitTimeout, serviceTimeout int) (*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateBasicInfo", ctx, id, name, detail, routingMethod, tagIDs, waitFlowID, waitTimeout, serviceTimeout)
	ret0, _ := ret[0].(*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateBasicInfo indicates an expected call of UpdateBasicInfo.
func (mr *MockQueueHandlerMockRecorder) UpdateBasicInfo(ctx, id, name, detail, routingMethod, tagIDs, waitFlowID, waitTimeout, serviceTimeout any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateBasicInfo", reflect.TypeOf((*MockQueueHandler)(nil).UpdateBasicInfo), ctx, id, name, detail, routingMethod, tagIDs, waitFlowID, waitTimeout, serviceTimeout)
}

// UpdateExecute mocks base method.
func (m *MockQueueHandler) UpdateExecute(ctx context.Context, id uuid.UUID, execute queue.Execute) (*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateExecute", ctx, id, execute)
	ret0, _ := ret[0].(*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateExecute indicates an expected call of UpdateExecute.
func (mr *MockQueueHandlerMockRecorder) UpdateExecute(ctx, id, execute any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateExecute", reflect.TypeOf((*MockQueueHandler)(nil).UpdateExecute), ctx, id, execute)
}

// UpdateRoutingMethod mocks base method.
func (m *MockQueueHandler) UpdateRoutingMethod(ctx context.Context, id uuid.UUID, routingMEthod queue.RoutingMethod) (*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateRoutingMethod", ctx, id, routingMEthod)
	ret0, _ := ret[0].(*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateRoutingMethod indicates an expected call of UpdateRoutingMethod.
func (mr *MockQueueHandlerMockRecorder) UpdateRoutingMethod(ctx, id, routingMEthod any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateRoutingMethod", reflect.TypeOf((*MockQueueHandler)(nil).UpdateRoutingMethod), ctx, id, routingMEthod)
}

// UpdateTagIDs mocks base method.
func (m *MockQueueHandler) UpdateTagIDs(ctx context.Context, id uuid.UUID, tagIDs []uuid.UUID) (*queue.Queue, error) {
	m.ctrl.T.Helper()
	ret := m.ctrl.Call(m, "UpdateTagIDs", ctx, id, tagIDs)
	ret0, _ := ret[0].(*queue.Queue)
	ret1, _ := ret[1].(error)
	return ret0, ret1
}

// UpdateTagIDs indicates an expected call of UpdateTagIDs.
func (mr *MockQueueHandlerMockRecorder) UpdateTagIDs(ctx, id, tagIDs any) *gomock.Call {
	mr.mock.ctrl.T.Helper()
	return mr.mock.ctrl.RecordCallWithMethodType(mr.mock, "UpdateTagIDs", reflect.TypeOf((*MockQueueHandler)(nil).UpdateTagIDs), ctx, id, tagIDs)
}

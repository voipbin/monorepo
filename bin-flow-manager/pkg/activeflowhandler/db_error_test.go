package activeflowhandler

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

type dbErrorMocks struct {
	util     *utilhandler.MockUtilHandler
	db       *dbhandler.MockDBHandler
	req      *requesthandler.MockRequestHandler
	notify   *notifyhandler.MockNotifyHandler
	stackmap *stackmaphandler.MockStackmapHandler
	variable *variablehandler.MockVariableHandler
}

// newDBErrorTestHandler returns a handler whose dependencies are strict mocks.
func newDBErrorTestHandler(mc *gomock.Controller) (*activeflowHandler, *dbErrorMocks) {
	m := &dbErrorMocks{
		util:     utilhandler.NewMockUtilHandler(mc),
		db:       dbhandler.NewMockDBHandler(mc),
		req:      requesthandler.NewMockRequestHandler(mc),
		notify:   notifyhandler.NewMockNotifyHandler(mc),
		stackmap: stackmaphandler.NewMockStackmapHandler(mc),
		variable: variablehandler.NewMockVariableHandler(mc),
	}

	h := &activeflowHandler{
		utilHandler:     m.util,
		db:              m.db,
		reqHandler:      m.req,
		notifyHandler:   m.notify,
		actionHandler:   actionhandler.NewMockActionHandler(mc),
		variableHandler: m.variable,
		stackmapHandler: m.stackmap,
	}

	return h, m
}

var (
	dbErrID         = uuid.FromStringOrNil("c1a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d2e01")
	dbErrCustomerID = uuid.FromStringOrNil("c1a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d2e02")
	dbErrFlowID     = uuid.FromStringOrNil("c1a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d2e03")
	dbErrActionID   = uuid.FromStringOrNil("c1a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d2e04")
	dbErrStackID    = uuid.FromStringOrNil("c1a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d2e05")
)

func dbErrActiveflow() *activeflow.Activeflow {
	return &activeflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         dbErrID,
			CustomerID: dbErrCustomerID,
		},
		Status:         activeflow.StatusRunning,
		ReferenceType:  activeflow.ReferenceTypeCall,
		CurrentStackID: stack.IDMain,
		CurrentAction:  action.Action{ID: dbErrActionID, Type: action.TypeAnswer},
	}
}

func Test_Create_errors(t *testing.T) {
	errDep := errors.New("dependency failure")
	otherCustomerID := uuid.FromStringOrNil("c1a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d2e99")

	tests := []struct {
		name string

		setup func(m *dbErrorMocks)

		expectErrPart string
		expectIs      error
	}{
		{
			name: "flow get fails",
			setup: func(m *dbErrorMocks) {
				m.req.EXPECT().FlowV1FlowGet(gomock.Any(), dbErrFlowID).Return(nil, errDep)
			},
			expectErrPart: "could not get flow",
			expectIs:      errDep,
		},
		{
			name: "flow belongs to another customer",
			setup: func(m *dbErrorMocks) {
				m.req.EXPECT().FlowV1FlowGet(gomock.Any(), dbErrFlowID).Return(&flow.Flow{
					Identity: commonidentity.Identity{ID: dbErrFlowID, CustomerID: otherCustomerID},
				}, nil)
			},
			expectErrPart: "the customer has no permission",
		},
		{
			name: "activeflow create fails",
			setup: func(m *dbErrorMocks) {
				m.req.EXPECT().FlowV1FlowGet(gomock.Any(), dbErrFlowID).Return(&flow.Flow{
					Identity: commonidentity.Identity{ID: dbErrFlowID, CustomerID: dbErrCustomerID},
				}, nil)
				m.stackmap.EXPECT().Create(gomock.Any()).Return(map[uuid.UUID]*stack.Stack{})
				m.db.EXPECT().ActiveflowCreate(gomock.Any(), gomock.Any()).Return(errDep)
			},
			expectErrPart: "could not create the active flow",
			expectIs:      errDep,
		},
		{
			name: "created activeflow can not be read back",
			setup: func(m *dbErrorMocks) {
				m.req.EXPECT().FlowV1FlowGet(gomock.Any(), dbErrFlowID).Return(&flow.Flow{
					Identity: commonidentity.Identity{ID: dbErrFlowID, CustomerID: dbErrCustomerID},
				}, nil)
				m.stackmap.EXPECT().Create(gomock.Any()).Return(map[uuid.UUID]*stack.Stack{})
				m.db.EXPECT().ActiveflowCreate(gomock.Any(), gomock.Any()).Return(nil)
				m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(nil, errDep)
			},
			expectErrPart: "could not get created active flow",
			expectIs:      errDep,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newDBErrorTestHandler(mc)
			tt.setup(m)

			res, err := h.Create(context.Background(), dbErrID, dbErrCustomerID, activeflow.ReferenceTypeCall, uuid.Nil, uuid.Nil, dbErrFlowID, nil, "", "")
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErrPart) {
				t.Fatalf("Wrong match. expect error containing %q, got: %v", tt.expectErrPart, err)
			}
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) {
				t.Errorf("Wrong match. expect cause: %v, got: %v", tt.expectIs, err)
			}
		})
	}
}

func Test_SetForwardActionID_errors(t *testing.T) {
	errDep := errors.New("dependency failure")

	tests := []struct {
		name string

		setup func(m *dbErrorMocks)

		expectErrPart string
		expectIs      error
	}{
		{
			name: "lock acquisition fails",
			setup: func(m *dbErrorMocks) {
				m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), dbErrID).Return(nil, errDep)
			},
			expectErrPart: "could not get activeflow",
			expectIs:      errDep,
		},
		{
			name: "forward action not found releases the lock",
			setup: func(m *dbErrorMocks) {
				m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), dbErrID).Return(dbErrActiveflow(), nil)
				m.stackmap.EXPECT().GetAction(gomock.Any(), stack.IDMain, dbErrActionID, false).Return(uuid.Nil, nil, errDep)
				m.db.EXPECT().ActiveflowReleaseLock(gomock.Any(), dbErrID).Return(nil)
			},
			expectErrPart: "forward action not found",
		},
		{
			name: "progress update fails releases the lock",
			setup: func(m *dbErrorMocks) {
				m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), dbErrID).Return(dbErrActiveflow(), nil)
				m.stackmap.EXPECT().GetAction(gomock.Any(), stack.IDMain, dbErrActionID, false).Return(dbErrStackID, &action.Action{ID: dbErrActionID}, nil)
				m.db.EXPECT().ActiveflowUpdate(gomock.Any(), dbErrID, gomock.Any()).Return(errDep)
				m.db.EXPECT().ActiveflowReleaseLock(gomock.Any(), dbErrID).Return(nil)
			},
			expectErrPart: "could not update the active flow",
			expectIs:      errDep,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newDBErrorTestHandler(mc)
			tt.setup(m)

			err := h.SetForwardActionID(context.Background(), dbErrID, dbErrActionID, false)
			if err == nil || !strings.Contains(err.Error(), tt.expectErrPart) {
				t.Fatalf("Wrong match. expect error containing %q, got: %v", tt.expectErrPart, err)
			}
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) {
				t.Errorf("Wrong match. expect cause: %v, got: %v", tt.expectIs, err)
			}
		})
	}
}

// Test_SetForwardActionID_forwardNow verifies that the immediate forward is sent
// to the call manager for a call reference, and that a failure of that request
// does not change the result of the call.
func Test_SetForwardActionID_forwardNow(t *testing.T) {
	errDep := errors.New("dependency failure")

	tests := []struct {
		name string

		referenceType activeflow.ReferenceType
		sendErr       error
		expectSend    bool
	}{
		{"call reference sends the request", activeflow.ReferenceTypeCall, nil, true},
		{"call reference send failure keeps the result", activeflow.ReferenceTypeCall, errDep, true},
		{"unsupported reference sends nothing", activeflow.ReferenceTypeConversation, nil, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newDBErrorTestHandler(mc)

			af := dbErrActiveflow()
			af.ReferenceType = tt.referenceType
			af.ReferenceID = uuid.FromStringOrNil("c1a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d2e06")

			m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), dbErrID).Return(af, nil)
			m.stackmap.EXPECT().GetAction(gomock.Any(), stack.IDMain, dbErrActionID, false).Return(dbErrStackID, &action.Action{ID: dbErrActionID}, nil)
			m.db.EXPECT().ActiveflowUpdate(gomock.Any(), dbErrID, gomock.Any()).Return(nil)
			m.db.EXPECT().ActiveflowReleaseLock(gomock.Any(), dbErrID).Return(nil)

			sent := make(chan struct{})
			if tt.expectSend {
				m.req.EXPECT().CallV1CallActionNext(gomock.Any(), af.ReferenceID, true).DoAndReturn(
					func(_ context.Context, _ uuid.UUID, _ bool) error {
						close(sent)
						return tt.sendErr
					})
			}

			if err := h.SetForwardActionID(context.Background(), dbErrID, dbErrActionID, true); err != nil {
				t.Fatalf("Wrong match. expect: ok, got: %v", err)
			}

			if tt.expectSend {
				select {
				case <-sent:
				case <-time.After(5 * time.Second):
					t.Fatalf("the action next request was not sent")
				}
			}
		})
	}
}

func Test_updateCurrentAction_errors(t *testing.T) {
	errDep := errors.New("dependency failure")

	tests := []struct {
		name string

		setup func(m *dbErrorMocks)

		expectErrPart string
	}{
		{
			name: "activeflow get fails",
			setup: func(m *dbErrorMocks) {
				m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(nil, errDep)
			},
			expectErrPart: "could not get activeflow",
		},
		{
			name: "update fails",
			setup: func(m *dbErrorMocks) {
				m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(dbErrActiveflow(), nil)
				m.db.EXPECT().ActiveflowUpdate(gomock.Any(), dbErrID, gomock.Any()).Return(errDep)
			},
			expectErrPart: "could not update the active flow",
		},
		{
			name: "updated activeflow can not be read back",
			setup: func(m *dbErrorMocks) {
				gomock.InOrder(
					m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(dbErrActiveflow(), nil),
					m.db.EXPECT().ActiveflowUpdate(gomock.Any(), dbErrID, gomock.Any()).Return(nil),
					m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(nil, errDep),
				)
			},
			expectErrPart: "could not get updated activeflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newDBErrorTestHandler(mc)
			tt.setup(m)

			res, err := h.updateCurrentAction(context.Background(), dbErrID, dbErrStackID, &action.Action{ID: dbErrActionID})
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErrPart) || !errors.Is(err, errDep) {
				t.Errorf("Wrong match. expect error containing %q caused by the dependency, got: %v", tt.expectErrPart, err)
			}
		})
	}
}

func Test_updateStatus_errors(t *testing.T) {
	errDep := errors.New("dependency failure")

	tests := []struct {
		name string

		setup func(m *dbErrorMocks)

		expectErrPart string
	}{
		{
			name: "update fails",
			setup: func(m *dbErrorMocks) {
				m.db.EXPECT().ActiveflowUpdate(gomock.Any(), dbErrID, map[activeflow.Field]any{activeflow.FieldStatus: activeflow.StatusEnded}).Return(errDep)
			},
			expectErrPart: "could not update activeflow status",
		},
		{
			name: "updated activeflow can not be read back",
			setup: func(m *dbErrorMocks) {
				m.db.EXPECT().ActiveflowUpdate(gomock.Any(), dbErrID, gomock.Any()).Return(nil)
				m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(nil, errDep)
			},
			expectErrPart: "could not get updated activeflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newDBErrorTestHandler(mc)
			tt.setup(m)

			res, err := h.updateStatus(context.Background(), dbErrID, activeflow.StatusEnded)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErrPart) || !errors.Is(err, errDep) {
				t.Errorf("Wrong match. expect error containing %q caused by the dependency, got: %v", tt.expectErrPart, err)
			}
		})
	}
}

func Test_updateNextAction_errors(t *testing.T) {
	errDep := errors.New("dependency failure")
	forwardActionID := uuid.FromStringOrNil("c1a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d2e07")

	tests := []struct {
		name string

		caID  uuid.UUID
		setup func(m *dbErrorMocks)

		expectErrPart string
		expectIs      error
	}{
		{
			name: "lock acquisition fails",
			caID: dbErrActionID,
			setup: func(m *dbErrorMocks) {
				m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), dbErrID).Return(nil, errDep)
			},
			expectErrPart: "could not get activeflow",
			expectIs:      errDep,
		},
		{
			name: "execute count exceeds the maximum",
			caID: dbErrActionID,
			setup: func(m *dbErrorMocks) {
				af := dbErrActiveflow()
				af.ExecuteCount = maxActiveFlowExecuteCount + 1
				m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), dbErrID).Return(af, nil)
				m.db.EXPECT().ActiveflowReleaseLock(gomock.Any(), dbErrID).Return(nil)
			},
			expectErrPart: fmt.Sprintf("exceed maximum action execution count. execute_count: %d", maxActiveFlowExecuteCount+1),
		},
		{
			name: "execute count at the maximum still passes the count check",
			caID: uuid.FromStringOrNil("c1a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d2e98"),
			setup: func(m *dbErrorMocks) {
				af := dbErrActiveflow()
				af.ExecuteCount = maxActiveFlowExecuteCount
				m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), dbErrID).Return(af, nil)
				m.db.EXPECT().ActiveflowReleaseLock(gomock.Any(), dbErrID).Return(nil)
			},
			// the request is rejected by the later current action id validation, not by the count check.
			expectErrPart: "could not pass the current action id validation",
		},
		{
			name: "activeflow already ended",
			caID: dbErrActionID,
			setup: func(m *dbErrorMocks) {
				af := dbErrActiveflow()
				af.Status = activeflow.StatusEnded
				m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), dbErrID).Return(af, nil)
				m.db.EXPECT().ActiveflowReleaseLock(gomock.Any(), dbErrID).Return(nil)
			},
			expectErrPart: "the activeflow ended",
		},
		{
			name: "current action id mismatch",
			caID: uuid.FromStringOrNil("c1a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d2e98"),
			setup: func(m *dbErrorMocks) {
				m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), dbErrID).Return(dbErrActiveflow(), nil)
				m.db.EXPECT().ActiveflowReleaseLock(gomock.Any(), dbErrID).Return(nil)
			},
			expectErrPart: "could not pass the current action id validation",
		},
		{
			name: "forward action lookup fails",
			caID: dbErrActionID,
			setup: func(m *dbErrorMocks) {
				af := dbErrActiveflow()
				af.ForwardStackID = dbErrStackID
				af.ForwardActionID = forwardActionID
				m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), dbErrID).Return(af, nil)
				m.stackmap.EXPECT().GetAction(gomock.Any(), dbErrStackID, forwardActionID, true).Return(uuid.Nil, nil, errDep)
				m.db.EXPECT().ActiveflowReleaseLock(gomock.Any(), dbErrID).Return(nil)
			},
			expectErrPart: "could not get action",
			expectIs:      errDep,
		},
		{
			name: "variable get fails",
			caID: dbErrActionID,
			setup: func(m *dbErrorMocks) {
				m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), dbErrID).Return(dbErrActiveflow(), nil)
				m.stackmap.EXPECT().GetNextAction(gomock.Any(), stack.IDMain, dbErrActionID, true).Return(stack.IDMain, &action.Action{ID: forwardActionID, Type: action.TypeTalk})
				m.variable.EXPECT().Get(gomock.Any(), dbErrID).Return(nil, errDep)
				m.db.EXPECT().ActiveflowReleaseLock(gomock.Any(), dbErrID).Return(nil)
			},
			expectErrPart: "could not get variables",
			expectIs:      errDep,
		},
		{
			name: "current action update fails",
			caID: dbErrActionID,
			setup: func(m *dbErrorMocks) {
				next := &action.Action{ID: forwardActionID, Type: action.TypeTalk, Option: map[string]any{"text": "hello"}}
				vars := &variable.Variable{ID: dbErrID}
				m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), dbErrID).Return(dbErrActiveflow(), nil)
				m.stackmap.EXPECT().GetNextAction(gomock.Any(), stack.IDMain, dbErrActionID, true).Return(stack.IDMain, next)
				m.variable.EXPECT().Get(gomock.Any(), dbErrID).Return(vars, nil)
				m.variable.EXPECT().SubstituteOption(gomock.Any(), next.Option, vars)
				m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(nil, errDep)
				m.db.EXPECT().ActiveflowReleaseLock(gomock.Any(), dbErrID).Return(nil)
			},
			expectErrPart: "could not update the current action",
			expectIs:      errDep,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newDBErrorTestHandler(mc)
			tt.setup(m)

			res, err := h.updateNextAction(context.Background(), dbErrID, tt.caID)
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErrPart) {
				t.Fatalf("Wrong match. expect error containing %q, got: %v", tt.expectErrPart, err)
			}
			if tt.expectIs != nil && !errors.Is(err, tt.expectIs) {
				t.Errorf("Wrong match. expect cause: %v, got: %v", tt.expectIs, err)
			}
		})
	}
}

func Test_Delete_errors(t *testing.T) {
	errDep := errors.New("dependency failure")
	deletedAt := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	tests := []struct {
		name string

		setup func(m *dbErrorMocks)

		expectErrPart string
		expectRes     *activeflow.Activeflow
	}{
		{
			name: "activeflow get fails",
			setup: func(m *dbErrorMocks) {
				m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(nil, errDep)
			},
			expectErrPart: "could not get activeflow",
		},
		{
			name: "already deleted returns the stored activeflow without changes",
			setup: func(m *dbErrorMocks) {
				af := dbErrActiveflow()
				af.TMDelete = &deletedAt
				m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(af, nil)
			},
			expectRes: func() *activeflow.Activeflow {
				af := dbErrActiveflow()
				af.TMDelete = &deletedAt
				return af
			}(),
		},
		{
			name: "stop fails for a running activeflow",
			setup: func(m *dbErrorMocks) {
				gomock.InOrder(
					m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(dbErrActiveflow(), nil),
					m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(nil, errDep),
				)
			},
			expectErrPart: "could not stop the activeflow",
		},
		{
			name: "database delete fails",
			setup: func(m *dbErrorMocks) {
				af := dbErrActiveflow()
				af.Status = activeflow.StatusEnded
				m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(af, nil)
				m.db.EXPECT().ActiveflowDelete(gomock.Any(), dbErrID).Return(errDep)
			},
			expectErrPart: "could not delete the activeflow",
		},
		{
			name: "deleted activeflow can not be read back",
			setup: func(m *dbErrorMocks) {
				af := dbErrActiveflow()
				af.Status = activeflow.StatusEnded
				gomock.InOrder(
					m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(af, nil),
					m.db.EXPECT().ActiveflowDelete(gomock.Any(), dbErrID).Return(nil),
					m.db.EXPECT().ActiveflowGet(gomock.Any(), dbErrID).Return(nil, errDep),
				)
			},
			expectErrPart: "could not get deleted activeflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newDBErrorTestHandler(mc)
			tt.setup(m)

			res, err := h.Delete(context.Background(), dbErrID)
			if tt.expectErrPart == "" {
				if err != nil {
					t.Fatalf("Wrong match. expect: ok, got: %v", err)
				}
				if res == nil || res.TMDelete == nil || !res.TMDelete.Equal(deletedAt) || res.ID != dbErrID {
					t.Errorf("Wrong match. expect: the stored deleted activeflow, got: %v", res)
				}
				return
			}

			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
			if err == nil || !strings.Contains(err.Error(), tt.expectErrPart) || !errors.Is(err, errDep) {
				t.Errorf("Wrong match. expect error containing %q caused by the dependency, got: %v", tt.expectErrPart, err)
			}
		})
	}
}

func Test_List_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	errDep := errors.New("dependency failure")
	h, m := newDBErrorTestHandler(mc)

	filters := map[activeflow.Field]any{activeflow.FieldCustomerID: dbErrCustomerID}
	m.db.EXPECT().ActiveflowList(gomock.Any(), "token", uint64(10), filters).Return(nil, errDep)

	res, err := h.List(context.Background(), "token", 10, filters)
	if res != nil {
		t.Errorf("Wrong match. expect: nil result, got: %v", res)
	}
	if err == nil || !strings.Contains(err.Error(), "could not get activeflows") || !errors.Is(err, errDep) {
		t.Errorf("Wrong match. expect: list error caused by the dependency, got: %v", err)
	}
}

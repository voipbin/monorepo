package activeflowhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/flow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

var (
	cov90RefActiveflowID = uuid.FromStringOrNil("c9c1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c01")
	cov90OnCompleteID    = uuid.FromStringOrNil("c9c1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c02")
)

func Test_variableCreate_failure_coverage90(t *testing.T) {
	ctx := context.Background()

	// referenceVariables is a variable map that carries a valid complete count.
	referenceVariables := func() *variable.Variable {
		return &variable.Variable{Variables: map[string]string{variableActiveflowCompleteCount: "0"}}
	}

	tests := []struct {
		name string

		initialVariables map[string]string
		setup            func(m *cov90Mocks, af *activeflow.Activeflow)

		expectMessage string
	}{
		{
			name: "the reference variable can not be read",
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.variable.EXPECT().Get(ctx, cov90RefActiveflowID).Return(nil, errDepInjected)
			},
			expectMessage: "could not set variable from reference activeflow",
		},
		{
			name: "the reference variable has no complete count",
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.variable.EXPECT().Get(ctx, cov90RefActiveflowID).Return(&variable.Variable{Variables: map[string]string{}}, nil)
			},
			expectMessage: "could not find the complete count variable",
		},
		{
			name: "the reference complete count is not a number",
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.variable.EXPECT().Get(ctx, cov90RefActiveflowID).Return(&variable.Variable{Variables: map[string]string{variableActiveflowCompleteCount: "many"}}, nil)
			},
			expectMessage: "could not parse the complete count variable",
		},
		{
			name: "the chain reached the maximum complete count",
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.variable.EXPECT().Get(ctx, cov90RefActiveflowID).Return(&variable.Variable{Variables: map[string]string{variableActiveflowCompleteCount: "4"}}, nil)
			},
			expectMessage: "reached the max complete count",
		},
		{
			name: "the variable can not be stored",
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.variable.EXPECT().Get(ctx, cov90RefActiveflowID).Return(referenceVariables(), nil)
				m.variable.EXPECT().Create(ctx, af.ID, gomock.Any()).Return(nil, errDepInjected)
			},
			expectMessage: "could not set the variable",
		},
		{
			// the inherited variables are so large that the merged map exceeds the cap while an
			// external variable was accepted, so the inheritance is read again and that read fails.
			name:             "the oversized merge reads the reference variable again and fails",
			initialVariables: map[string]string{"customer_key": "value"},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				large := referenceVariables()
				large.Variables["bulk"] = strings.Repeat("x", mergedVariablesMaxTotalBytes+1)
				gomock.InOrder(
					m.variable.EXPECT().Get(ctx, cov90RefActiveflowID).Return(large, nil),
					m.variable.EXPECT().Get(ctx, cov90RefActiveflowID).Return(nil, errDepInjected),
				)
			},
			expectMessage: "could not re-set variable from reference activeflow",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newCov90Handler(mc)
			af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)
			af.ReferenceActiveflowID = cov90RefActiveflowID
			tt.setup(m, af)

			res, err := h.variableCreate(ctx, af, tt.initialVariables)
			if err == nil || !strings.Contains(err.Error(), tt.expectMessage) {
				t.Fatalf("Wrong error.\nexpect to contain: %s\ngot: %v", tt.expectMessage, err)
			}
			if res != nil {
				t.Errorf("no variable is expected with the error. got: %v", res)
			}
		})
	}
}

func Test_sanitizeInitialVariables_onlyDroppedKeys_coverage90(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _ := newCov90Handler(mc)

	// every key is either blank or in the reserved namespace, so nothing survives.
	res := h.sanitizeInitialVariables(map[string]string{
		"   ":                           "blank key",
		"voipbin.activeflow.id":         "reserved",
		"  VoIPBin.activeflow.flow_id ": "reserved with a different case",
	})
	if len(res) != 0 {
		t.Errorf("Nothing must survive. got: %v", res)
	}
}

func Test_compareCondition_unknownCondition_coverage90(t *testing.T) {
	// an unregistered condition never matches, for any pair of values.
	if compareCondition[int]("unknown", 1, 1) {
		t.Errorf("An unknown condition must not match equal values")
	}
	if compareCondition[string]("unknown", "a", "b") {
		t.Errorf("An unknown condition must not match different values")
	}
}

func Test_actionGetsFromFlow_coverage90(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("c9a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e02")
	otherCustomerID := uuid.FromStringOrNil("c9a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e99")

	tests := []struct {
		name string

		customerID uuid.UUID
		flowCustom uuid.UUID

		expectErr string
	}{
		{"flow of another customer is refused", customerID, otherCustomerID, "the customer has no permission"},
		{"flow of the same customer is returned", customerID, customerID, ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newCov90Handler(mc)
			f := cov90Flow
			f.CustomerID = tt.flowCustom
			f.Actions = cov90Service.PushActions
			m.req.EXPECT().FlowV1FlowGet(ctx, cov90ActionID).Return(&f, nil)

			res, err := h.actionGetsFromFlow(ctx, cov90ActionID, tt.customerID)
			if tt.expectErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.expectErr) {
					t.Fatalf("Wrong error.\nexpect to contain: %s\ngot: %v", tt.expectErr, err)
				}
				if res != nil {
					t.Errorf("no actions are expected with the error. got: %v", res)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(res) != 1 || res[0].ID != cov90ActionID {
				t.Errorf("The actions of the flow must be returned. got: %v", res)
			}
		})
	}
}

func Test_EventCustomerDeleted_failure_coverage90(t *testing.T) {
	ctx := context.Background()
	customerID := uuid.FromStringOrNil("c9a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e02")
	cu := &cmcustomer.Customer{}
	cu.ID = customerID

	t.Run("the list failure is reported", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		m.util.EXPECT().TimeGetCurTime().Return("2020-04-18 03:22:17.995000")
		m.db.EXPECT().ActiveflowList(ctx, "2020-04-18 03:22:17.995000", uint64(1000), gomock.Any()).Return(nil, errDepInjected)

		err := h.EventCustomerDeleted(ctx, cu)
		if err == nil || !strings.Contains(err.Error(), "could not get activeflows list") {
			t.Errorf("expected the list failure, got: %v", err)
		}
	})

	t.Run("a delete failure does not stop the others", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		first := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)
		second := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)
		second.ID = uuid.FromStringOrNil("c9a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e55")

		m.util.EXPECT().TimeGetCurTime().Return("2020-04-18 03:22:17.995000")
		m.db.EXPECT().ActiveflowList(ctx, gomock.Any(), uint64(1000), gomock.Any()).Return([]*activeflow.Activeflow{first, second}, nil)

		// the first one can not be read, the second one is already deleted.
		m.db.EXPECT().ActiveflowGet(ctx, first.ID).Return(nil, errDepInjected)
		now := time.Date(2020, 4, 18, 3, 22, 17, 0, time.UTC)
		second.TMDelete = &now
		m.db.EXPECT().ActiveflowGet(ctx, second.ID).Return(second, nil)

		if err := h.EventCustomerDeleted(ctx, cu); err != nil {
			t.Errorf("A delete failure must not fail the event. got: %v", err)
		}
	})
}

func Test_Stop_coverage90(t *testing.T) {
	ctx := context.Background()

	t.Run("an activeflow that has a creation time is stopped", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)
		created := time.Now().Add(-time.Minute)
		af.TMCreate = &created

		m.db.EXPECT().ActiveflowGet(ctx, af.ID).Return(af, nil).Times(2)
		m.db.EXPECT().ActiveflowUpdate(ctx, af.ID, map[activeflow.Field]any{activeflow.FieldStatus: activeflow.StatusEnded}).Return(nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, af.CustomerID, activeflow.EventTypeActiveflowUpdated, af)

		res, err := h.Stop(ctx, af.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != af {
			t.Errorf("The stopped activeflow must be returned. got: %v", res)
		}
	})

	t.Run("a failure of the on complete flow does not fail the stop", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)
		af.OnCompleteFlowID = cov90OnCompleteID

		m.db.EXPECT().ActiveflowGet(ctx, af.ID).Return(af, nil).Times(2)
		m.db.EXPECT().ActiveflowUpdate(ctx, af.ID, gomock.Any()).Return(nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, af.CustomerID, activeflow.EventTypeActiveflowUpdated, af)
		// the on complete activeflow is created with a generated id, and the flow lookup fails.
		m.util.EXPECT().UUIDCreate().Return(cov90ActionID)
		m.req.EXPECT().FlowV1FlowGet(ctx, cov90OnCompleteID).Return(nil, errDepInjected)

		res, err := h.Stop(ctx, af.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != af {
			t.Errorf("The stopped activeflow must be returned. got: %v", res)
		}
	})
}

func Test_startOnCompleteFlow_coverage90(t *testing.T) {
	ctx := context.Background()

	t.Run("the creation failure is reported", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)
		af.OnCompleteFlowID = cov90OnCompleteID

		m.util.EXPECT().UUIDCreate().Return(cov90ActionID)
		m.req.EXPECT().FlowV1FlowGet(ctx, cov90OnCompleteID).Return(nil, errDepInjected)

		res, err := h.startOnCompleteFlow(ctx, af)
		if err == nil || !strings.Contains(err.Error(), "could not create on complete activeflow") {
			t.Fatalf("expected the creation failure, got: %v", err)
		}
		if !errors.Is(err, errDepInjected) {
			t.Errorf("The cause must be kept. got: %v", err)
		}
		if res != nil {
			t.Errorf("no activeflow is expected with the error. got: %v", res)
		}
	})

	t.Run("a failure of the execution request does not fail the start", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)
		af.OnCompleteFlowID = cov90OnCompleteID

		created := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)
		created.ID = cov90ActionID
		created.ReferenceActiveflowID = af.ID

		m.util.EXPECT().UUIDCreate().Return(cov90ActionID)
		m.req.EXPECT().FlowV1FlowGet(ctx, cov90OnCompleteID).Return(&flow.Flow{
			Identity: cov90Flow.Identity,
			Actions:  cov90Service.PushActions,
		}, nil)
		m.stack.EXPECT().Create(cov90Service.PushActions).Return(map[uuid.UUID]*stack.Stack{})
		m.db.EXPECT().ActiveflowCreate(ctx, gomock.Any()).Return(nil)
		m.db.EXPECT().ActiveflowGet(ctx, cov90ActionID).Return(created, nil)
		// the variables of the on complete activeflow are inherited from the stopped one, and the read fails.
		m.variable.EXPECT().Get(ctx, af.ID).Return(nil, errDepInjected)
		m.notify.EXPECT().PublishWebhookEvent(ctx, created.CustomerID, activeflow.EventTypeActiveflowCreated, gomock.Any())

		executed := make(chan struct{})
		m.req.EXPECT().FlowV1ActiveflowExecute(gomock.Any(), created.ID).DoAndReturn(
			func(_ context.Context, _ uuid.UUID) error {
				defer close(executed)
				return errDepInjected
			})

		res, err := h.startOnCompleteFlow(ctx, af)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != created {
			t.Errorf("The created activeflow must be returned. got: %v", res)
		}

		select {
		case <-executed:
		case <-time.After(5 * time.Second):
			t.Fatalf("The execution request was not sent")
		}
	})
}

func Test_Delete_coverage90(t *testing.T) {
	ctx := context.Background()

	t.Run("an activeflow that is not ended is stopped before the delete", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)
		af.Status = activeflow.StatusRunning

		// the reads are the delete's, the stop's, the status update's and the delete's result.
		m.db.EXPECT().ActiveflowGet(ctx, af.ID).Return(af, nil).Times(4)
		m.db.EXPECT().ActiveflowUpdate(ctx, af.ID, map[activeflow.Field]any{activeflow.FieldStatus: activeflow.StatusEnded}).Return(nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, af.CustomerID, activeflow.EventTypeActiveflowUpdated, af)
		m.db.EXPECT().ActiveflowDelete(ctx, af.ID).Return(nil)
		m.notify.EXPECT().PublishWebhookEvent(ctx, af.CustomerID, activeflow.EventTypeActiveflowDeleted, af)

		res, err := h.Delete(ctx, af.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != af {
			t.Errorf("The deleted activeflow must be returned. got: %v", res)
		}
	})
}

func Test_NewActiveflowHandler_coverage90(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockAction := actionhandler.NewMockActionHandler(mc)
	mockVariable := variablehandler.NewMockVariableHandler(mc)

	res := NewActiveflowHandler(mockDB, mockReq, mockNotify, mockAction, mockVariable)
	h, ok := res.(*activeflowHandler)
	if !ok {
		t.Fatalf("Wrong type. got: %T", res)
	}

	if h.db != mockDB || h.reqHandler != mockReq || h.notifyHandler != mockNotify || h.actionHandler != mockAction || h.variableHandler != mockVariable {
		t.Errorf("The given dependencies must be kept")
	}
	if h.utilHandler == nil || h.stackmapHandler == nil {
		t.Errorf("The own dependencies must be created. util: %v, stackmap: %v", h.utilHandler, h.stackmapHandler)
	}
}

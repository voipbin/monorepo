package activeflowhandler

import (
	"context"
	"errors"
	"testing"

	cerrors "monorepo/bin-common-handler/models/errors"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
)

var errAfSynthetic = errors.New("synthetic failure")

type afErrMocks struct {
	db     *dbhandler.MockDBHandler
	stack  *stackmaphandler.MockStackmapHandler
	action *actionhandler.MockActionHandler
}

func newAfErrHandler(mc *gomock.Controller) (*activeflowHandler, *afErrMocks) {
	m := &afErrMocks{
		db:     dbhandler.NewMockDBHandler(mc),
		stack:  stackmaphandler.NewMockStackmapHandler(mc),
		action: actionhandler.NewMockActionHandler(mc),
	}
	return &activeflowHandler{db: m.db, stackmapHandler: m.stack, actionHandler: m.action}, m
}

var (
	afErrID      = uuid.FromStringOrNil("a508739b-d98d-40fb-8a47-61e9a70958cd")
	afErrStackID = uuid.FromStringOrNil("2b3b0a0e-1c2d-11ee-9a1d-0f3c8d1ba001")
	afErrActID   = uuid.FromStringOrNil("2b3b0a0e-1c2d-11ee-9a1d-0f3c8d1ba002")
)

func afErrActiveflow(stackID uuid.UUID) *activeflow.Activeflow {
	af := &activeflow.Activeflow{
		CurrentStackID: stackID,
		CurrentAction:  action.Action{ID: afErrActID},
		StackMap:       map[uuid.UUID]*stack.Stack{stackID: {ID: stackID}},
	}
	af.ID = afErrID
	return af
}

func Test_Get_notFoundIsTyped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newAfErrHandler(mc)
	m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(nil, dbhandler.ErrNotFound)

	res, err := h.Get(context.Background(), afErrID)
	if res != nil {
		t.Errorf("Wrong result. expect: nil, got: %v", res)
	}

	var ve *cerrors.VoipbinError
	if !errors.As(err, &ve) {
		t.Fatalf("Wrong error type. expect: VoipbinError, got: %v", err)
	}
	if ve.Status != cerrors.StatusNotFound || ve.Reason != "ACTIVEFLOW_NOT_FOUND" {
		t.Errorf("Wrong error. expect: NOT_FOUND/ACTIVEFLOW_NOT_FOUND, got: %v/%s", ve.Status, ve.Reason)
	}
	if !errors.Is(err, dbhandler.ErrNotFound) {
		t.Errorf("The original cause is lost. got: %v", err)
	}
}

func Test_Get_otherErrorKeepsCause(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newAfErrHandler(mc)
	m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(nil, errAfSynthetic)

	_, err := h.Get(context.Background(), afErrID)
	var ve *cerrors.VoipbinError
	if errors.As(err, &ve) {
		t.Errorf("A generic failure must not be reported as a typed error. got: %v", ve)
	}
	if !errors.Is(err, errAfSynthetic) {
		t.Errorf("Wrong error. expect: %v, got: %v", errAfSynthetic, err)
	}
}

func Test_ServiceStop_error(t *testing.T) {
	tests := []struct {
		name      string
		serviceID uuid.UUID
		setup     func(m *afErrMocks)
	}{
		{
			name:      "activeflow lookup fails",
			serviceID: afErrStackID,
			setup: func(m *afErrMocks) {
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(nil, errAfSynthetic)
			},
		},
		{
			name:      "service is not in the current stack and nothing is popped",
			serviceID: afErrActID,
			setup: func(m *afErrMocks) {
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(afErrActiveflow(afErrStackID), nil)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newAfErrHandler(mc)
			tt.setup(m)

			if err := h.ServiceStop(context.Background(), afErrID, tt.serviceID); err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
		})
	}
}

func Test_PushActions_error(t *testing.T) {
	actions := []action.Action{{Type: action.TypeAnswer}}
	flowActions := []action.Action{{ID: afErrActID, Type: action.TypeAnswer}}

	tests := []struct {
		name        string
		setup       func(m *afErrMocks)
		expectCause error
	}{
		{
			name: "activeflow lookup fails",
			setup: func(m *afErrMocks) {
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(nil, errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
		{
			name: "action generation fails and the stack is not touched",
			setup: func(m *afErrMocks) {
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(afErrActiveflow(afErrStackID), nil)
				m.action.EXPECT().GenerateFlowActions(gomock.Any(), actions).Return(nil, errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
		{
			name: "pushing the stack fails",
			setup: func(m *afErrMocks) {
				af := afErrActiveflow(afErrStackID)
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(af, nil)
				m.action.EXPECT().GenerateFlowActions(gomock.Any(), actions).Return(flowActions, nil)
				m.stack.EXPECT().PushStackByActions(af.StackMap, uuid.Nil, flowActions, afErrStackID, afErrActID).Return(nil, errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
		{
			name: "reading the updated activeflow fails",
			setup: func(m *afErrMocks) {
				af := afErrActiveflow(afErrStackID)
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(af, nil)
				m.action.EXPECT().GenerateFlowActions(gomock.Any(), actions).Return(flowActions, nil)
				m.stack.EXPECT().PushStackByActions(af.StackMap, uuid.Nil, flowActions, afErrStackID, afErrActID).Return(&stack.Stack{ID: afErrStackID, Actions: flowActions}, nil)
				m.db.EXPECT().ActiveflowUpdate(gomock.Any(), afErrID, gomock.Any()).Return(nil)
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(nil, errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newAfErrHandler(mc)
			tt.setup(m)

			res, err := h.PushActions(context.Background(), afErrID, actions)
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
			if !errors.Is(err, tt.expectCause) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectCause, err)
			}
		})
	}
}

func Test_AddActions_error(t *testing.T) {
	actions := []action.Action{{Type: action.TypeHangup}}
	flowActions := []action.Action{{ID: afErrActID, Type: action.TypeHangup}}

	tests := []struct {
		name        string
		setup       func(m *afErrMocks)
		expectCause error // nil means a plain error without a wrapped cause
	}{
		{
			name: "action generation fails before the activeflow is read",
			setup: func(m *afErrMocks) {
				m.action.EXPECT().GenerateFlowActions(gomock.Any(), actions).Return(nil, errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
		{
			name: "activeflow lookup fails",
			setup: func(m *afErrMocks) {
				m.action.EXPECT().GenerateFlowActions(gomock.Any(), actions).Return(flowActions, nil)
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(nil, errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
		{
			name: "adding to the main stack is rejected without any write",
			setup: func(m *afErrMocks) {
				m.action.EXPECT().GenerateFlowActions(gomock.Any(), actions).Return(flowActions, nil)
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(afErrActiveflow(stack.IDMain), nil)
			},
		},
		{
			name: "stackmap rejects the actions",
			setup: func(m *afErrMocks) {
				af := afErrActiveflow(afErrStackID)
				m.action.EXPECT().GenerateFlowActions(gomock.Any(), actions).Return(flowActions, nil)
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(af, nil)
				m.stack.EXPECT().AddActions(af.StackMap, afErrStackID, afErrActID, flowActions).Return(errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
		{
			name: "persisting the progress fails",
			setup: func(m *afErrMocks) {
				af := afErrActiveflow(afErrStackID)
				m.action.EXPECT().GenerateFlowActions(gomock.Any(), actions).Return(flowActions, nil)
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(af, nil)
				m.stack.EXPECT().AddActions(af.StackMap, afErrStackID, afErrActID, flowActions).Return(nil)
				m.db.EXPECT().ActiveflowUpdate(gomock.Any(), afErrID, gomock.Any()).Return(errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
		{
			name: "reading the updated activeflow fails",
			setup: func(m *afErrMocks) {
				af := afErrActiveflow(afErrStackID)
				m.action.EXPECT().GenerateFlowActions(gomock.Any(), actions).Return(flowActions, nil)
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(af, nil)
				m.stack.EXPECT().AddActions(af.StackMap, afErrStackID, afErrActID, flowActions).Return(nil)
				m.db.EXPECT().ActiveflowUpdate(gomock.Any(), afErrID, gomock.Any()).Return(nil)
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(nil, errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newAfErrHandler(mc)
			tt.setup(m)

			res, err := h.AddActions(context.Background(), afErrID, actions)
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if tt.expectCause != nil && !errors.Is(err, tt.expectCause) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectCause, err)
			}
		})
	}
}

func Test_PopStackWithStackID_error(t *testing.T) {
	returnStackID := uuid.FromStringOrNil("2b3b0a0e-1c2d-11ee-9a1d-0f3c8d1ba003")
	returnActionID := uuid.FromStringOrNil("2b3b0a0e-1c2d-11ee-9a1d-0f3c8d1ba004")

	tests := []struct {
		name        string
		stackID     uuid.UUID
		setup       func(m *afErrMocks, af *activeflow.Activeflow)
		expectCause error
	}{
		{
			name:    "stack id differs from the current stack and nothing is popped",
			stackID: afErrActID,
			setup:   func(m *afErrMocks, af *activeflow.Activeflow) {},
		},
		{
			name:    "stackmap cannot pop the stack",
			stackID: afErrStackID,
			setup: func(m *afErrMocks, af *activeflow.Activeflow) {
				m.stack.EXPECT().PopStack(af.StackMap, afErrStackID).Return(nil, errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
		{
			name:    "persisting the progress fails after the pop",
			stackID: afErrStackID,
			setup: func(m *afErrMocks, af *activeflow.Activeflow) {
				m.stack.EXPECT().PopStack(af.StackMap, afErrStackID).Return(&stack.Stack{ID: afErrStackID, ReturnStackID: returnStackID, ReturnActionID: returnActionID}, nil)
				m.stack.EXPECT().GetNextAction(af.StackMap, returnStackID, returnActionID, true).Return(returnStackID, &action.Action{ID: returnActionID})
				m.db.EXPECT().ActiveflowUpdate(gomock.Any(), afErrID, gomock.Any()).Return(errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newAfErrHandler(mc)
			af := afErrActiveflow(afErrStackID)
			tt.setup(m, af)

			err := h.PopStackWithStackID(context.Background(), af, tt.stackID)
			if err == nil {
				t.Fatalf("Wrong match. expect: error, got: nil")
			}
			if tt.expectCause != nil && !errors.Is(err, tt.expectCause) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectCause, err)
			}
		})
	}
}

func Test_PopStack_error(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newAfErrHandler(mc)
	af := afErrActiveflow(afErrStackID)
	m.stack.EXPECT().PopStack(af.StackMap, afErrStackID).Return(nil, errAfSynthetic)

	if err := h.PopStack(context.Background(), af); !errors.Is(err, errAfSynthetic) {
		t.Errorf("Wrong error. expect: %v, got: %v", errAfSynthetic, err)
	}
}

func Test_PushStack_error(t *testing.T) {
	actions := []action.Action{{ID: afErrActID, Type: action.TypeAnswer}}

	tests := []struct {
		name        string
		actions     []action.Action
		setup       func(m *afErrMocks, af *activeflow.Activeflow)
		expectError bool
		expectCause error
	}{
		{
			name:    "no actions means nothing to push",
			actions: nil,
			setup:   func(m *afErrMocks, af *activeflow.Activeflow) {},
		},
		{
			name:    "stackmap rejects the push",
			actions: actions,
			setup: func(m *afErrMocks, af *activeflow.Activeflow) {
				m.stack.EXPECT().PushStackByActions(af.StackMap, uuid.Nil, actions, afErrStackID, afErrActID).Return(nil, errAfSynthetic)
			},
			expectError: true,
			expectCause: errAfSynthetic,
		},
		{
			name:    "persisting the progress fails",
			actions: actions,
			setup: func(m *afErrMocks, af *activeflow.Activeflow) {
				m.stack.EXPECT().PushStackByActions(af.StackMap, uuid.Nil, actions, afErrStackID, afErrActID).Return(&stack.Stack{ID: afErrStackID, Actions: actions}, nil)
				m.db.EXPECT().ActiveflowUpdate(gomock.Any(), afErrID, gomock.Any()).Return(errAfSynthetic)
			},
			expectError: true,
			expectCause: errAfSynthetic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newAfErrHandler(mc)
			af := afErrActiveflow(afErrStackID)
			tt.setup(m, af)

			err := h.PushStack(context.Background(), af, uuid.Nil, tt.actions)
			if !tt.expectError {
				if err != nil {
					t.Errorf("Wrong match. expect: ok, got: %v", err)
				}
				return
			}
			if !errors.Is(err, tt.expectCause) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectCause, err)
			}
		})
	}
}

func Test_Stop_error(t *testing.T) {
	tests := []struct {
		name        string
		setup       func(m *afErrMocks)
		expectCause error
	}{
		{
			name: "activeflow lookup fails",
			setup: func(m *afErrMocks) {
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(nil, errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
		{
			name: "status update fails",
			setup: func(m *afErrMocks) {
				m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(&activeflow.Activeflow{Status: activeflow.StatusRunning}, nil)
				m.db.EXPECT().ActiveflowUpdate(gomock.Any(), afErrID, map[activeflow.Field]any{activeflow.FieldStatus: activeflow.StatusEnded}).Return(errAfSynthetic)
			},
			expectCause: errAfSynthetic,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newAfErrHandler(mc)
			tt.setup(m)

			res, err := h.Stop(context.Background(), afErrID)
			if res != nil {
				t.Errorf("Wrong result. expect: nil, got: %v", res)
			}
			if !errors.Is(err, tt.expectCause) {
				t.Errorf("Wrong error. expect: %v, got: %v", tt.expectCause, err)
			}
		})
	}
}

// Test_Stop_alreadyEnded verifies that stopping an ended activeflow returns it as is and writes nothing.
func Test_Stop_alreadyEnded(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newAfErrHandler(mc)
	ended := &activeflow.Activeflow{Status: activeflow.StatusEnded}
	m.db.EXPECT().ActiveflowGet(gomock.Any(), afErrID).Return(ended, nil)

	res, err := h.Stop(context.Background(), afErrID)
	if err != nil {
		t.Fatalf("Wrong match. expect: ok, got: %v", err)
	}
	if res != ended {
		t.Errorf("Wrong result. expect the same activeflow, got: %v", res)
	}
}

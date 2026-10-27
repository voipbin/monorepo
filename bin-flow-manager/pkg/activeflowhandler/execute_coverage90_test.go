package activeflowhandler

import (
	"context"
	"strings"
	"testing"

	cmcall "monorepo/bin-call-manager/models/call"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/models/service"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	mmmessage "monorepo/bin-message-manager/models/message"

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

// cov90Mocks holds the strict mocks of the handler built by newCov90Handler.
type cov90Mocks struct {
	req      *requesthandler.MockRequestHandler
	stack    *stackmaphandler.MockStackmapHandler
	db       *dbhandler.MockDBHandler
	variable *variablehandler.MockVariableHandler
	action   *actionhandler.MockActionHandler
	util     *utilhandler.MockUtilHandler
	notify   *notifyhandler.MockNotifyHandler
}

func newCov90Handler(mc *gomock.Controller) (*activeflowHandler, *cov90Mocks) {
	m := &cov90Mocks{
		req:      requesthandler.NewMockRequestHandler(mc),
		stack:    stackmaphandler.NewMockStackmapHandler(mc),
		db:       dbhandler.NewMockDBHandler(mc),
		variable: variablehandler.NewMockVariableHandler(mc),
		action:   actionhandler.NewMockActionHandler(mc),
		util:     utilhandler.NewMockUtilHandler(mc),
		notify:   notifyhandler.NewMockNotifyHandler(mc),
	}

	h := &activeflowHandler{
		utilHandler:     m.util,
		db:              m.db,
		reqHandler:      m.req,
		notifyHandler:   m.notify,
		actionHandler:   m.action,
		variableHandler: m.variable,
		stackmapHandler: m.stack,
	}

	return h, m
}

func newCov90Activeflow(refType activeflow.ReferenceType, actionType action.Type, option map[string]any) *activeflow.Activeflow {
	return &activeflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("c9a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e01"),
			CustomerID: uuid.FromStringOrNil("c9a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e02"),
		},
		ReferenceType:  refType,
		ReferenceID:    uuid.FromStringOrNil("c9a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e04"),
		CurrentStackID: stack.IDMain,
		CurrentAction: action.Action{
			ID:     uuid.FromStringOrNil("c9a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e03"),
			Type:   actionType,
			Option: option,
		},
	}
}

var (
	cov90StackID  = uuid.FromStringOrNil("c9b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c01")
	cov90ActionID = uuid.FromStringOrNil("c9b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c02")
	cov90Service  = &service.Service{
		ID:          uuid.FromStringOrNil("c9b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c03"),
		PushActions: []action.Action{{ID: cov90ActionID, Type: action.TypeAnswer}},
	}
)

var cov90Flow = flow.Flow{
	Identity: commonidentity.Identity{
		ID:         uuid.FromStringOrNil("c9b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c20"),
		CustomerID: uuid.FromStringOrNil("c9a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e02"),
	},
}

// expectCov90PushOK registers the stack push and the progress update that follow a
// successful PushStack.
func expectCov90PushOK(m *cov90Mocks, af *activeflow.Activeflow, stackID uuid.UUID, actions []action.Action) {
	m.stack.EXPECT().PushStackByActions(af.StackMap, stackID, actions, af.CurrentStackID, af.CurrentAction.ID).
		Return(&stack.Stack{ID: cov90StackID, Actions: []action.Action{{ID: cov90ActionID}}}, nil)
	m.db.EXPECT().ActiveflowUpdate(gomock.Any(), af.ID, gomock.Any()).Return(nil)
}

// Test_executeAction_handlerSuccess_coverage90 runs the dispatch of executeAction for the
// actions whose handler finishes without an error.
func Test_executeAction_handlerSuccess_coverage90(t *testing.T) {
	ctx := gomock.Any()

	tests := []struct {
		name string

		refType    activeflow.ReferenceType
		actionType action.Type
		option     map[string]any
		setup      func(m *cov90Mocks, af *activeflow.Activeflow)

		expectRes *action.Action
		// expectForwarded is true when the handler must point the activeflow to the pushed actions.
		expectForwarded bool
	}{
		{
			name:       "ai_summary pushes the actions of the service",
			refType:    activeflow.ReferenceTypeConversation,
			actionType: action.TypeAISummary,
			option:     map[string]any{},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().AIV1ServiceTypeSummaryStart(ctx, af.CustomerID, af.ID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(cov90Service, nil)
				expectCov90PushOK(m, af, cov90Service.ID, cov90Service.PushActions)
			},
			expectRes:       &action.ActionNext,
			expectForwarded: true,
		},
		{
			name:       "ai_talk on a call pushes the actions of the service",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeAITalk,
			option:     map[string]any{},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().AIV1ServiceTypeAIcallStart(ctx, gomock.Any(), gomock.Any(), af.ID, gomock.Any(), af.ReferenceID, gomock.Any()).Return(cov90Service, nil)
				expectCov90PushOK(m, af, cov90Service.ID, cov90Service.PushActions)
			},
			expectRes:       &action.ActionNext,
			expectForwarded: true,
		},
		{
			name:       "ai_talk on a conversation pushes the actions of the service",
			refType:    activeflow.ReferenceTypeConversation,
			actionType: action.TypeAITalk,
			option:     map[string]any{},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().AIV1ServiceTypeAIcallStart(ctx, gomock.Any(), gomock.Any(), af.ID, gomock.Any(), af.ReferenceID, gomock.Any()).Return(cov90Service, nil)
				expectCov90PushOK(m, af, cov90Service.ID, cov90Service.PushActions)
			},
			expectRes:       &action.ActionNext,
			expectForwarded: true,
		},
		{
			name:       "ai_task pushes the actions of the service",
			refType:    activeflow.ReferenceTypeConversation,
			actionType: action.TypeAITask,
			option:     map[string]any{},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().AIV1ServiceTypeTaskStart(ctx, gomock.Any(), gomock.Any(), af.ID).Return(cov90Service, nil)
				expectCov90PushOK(m, af, cov90Service.ID, cov90Service.PushActions)
			},
			expectRes:       &action.ActionNext,
			expectForwarded: true,
		},
		{
			name:       "block waits without calling any dependency",
			refType:    activeflow.ReferenceTypeConversation,
			actionType: action.TypeBlock,
			setup:      func(m *cov90Mocks, af *activeflow.Activeflow) {},
			expectRes:  &action.ActionEmpty,
		},
		{
			name:       "branch moves to the default target",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeBranch,
			option:     map[string]any{"default_target_id": cov90ActionID.String()},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.variable.EXPECT().Get(ctx, af.ID).Return(&variable.Variable{Variables: map[string]string{}}, nil)
				m.variable.EXPECT().SetVariable(ctx, af.ID, gomock.Any()).Return(nil)
				m.stack.EXPECT().GetAction(af.StackMap, af.CurrentStackID, cov90ActionID, false).Return(cov90StackID, &action.Action{ID: cov90ActionID}, nil)
				m.db.EXPECT().ActiveflowUpdate(ctx, af.ID, gomock.Any()).Return(nil)
			},
			expectRes:       &action.ActionNext,
			expectForwarded: true,
		},
		{
			name:       "call creates the outgoing call with the given flow",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeCall,
			option:     map[string]any{"flow_id": cov90ActionID.String()},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallsCreate(ctx, af.CustomerID, cov90ActionID, uuid.Nil, gomock.Any(), gomock.Any(), false, false, gomock.Any(), nil, nil).Return([]*cmcall.Call{}, nil, nil)
			},
			expectRes: &action.ActionNext,
		},
		{
			name:       "condition_call_digits passes with enough digits",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionCallDigits,
			option:     map[string]any{"length": 3},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallGetDigits(ctx, af.ReferenceID).Return("123", nil)
			},
			expectRes: &action.ActionNext,
		},
		{
			name:       "condition_call_status passes with the same status",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionCallStatus,
			option:     map[string]any{"status": string(cmcall.StatusProgressing)},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallGet(ctx, af.ReferenceID).Return(&cmcall.Call{Status: cmcall.StatusProgressing}, nil)
			},
			expectRes: &action.ActionNext,
		},
		{
			// the negative minute and the absent weekday and month skip every time based check.
			name:       "condition_datetime passes without any time based check",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionDatetime,
			option:     map[string]any{"minute": -1, "hour": -1},
			setup:      func(m *cov90Mocks, af *activeflow.Activeflow) {},
			expectRes:  &action.ActionNext,
		},
		{
			name:       "condition_variable passes with equal strings",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionVariable,
			option:     map[string]any{"condition": "==", "variable": "abc", "value_type": string(action.OptionConditionVariableTypeString), "value_string": "abc"},
			setup:      func(m *cov90Mocks, af *activeflow.Activeflow) {},
			expectRes:  &action.ActionNext,
		},
		{
			name:       "conference_join pushes the actions of the service",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConferenceJoin,
			option:     map[string]any{"conference_id": cov90ActionID.String()},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().ConferenceV1ServiceTypeConferencecallStart(ctx, af.ID, cov90ActionID, gomock.Any(), af.ReferenceID).Return(cov90Service, nil)
				expectCov90PushOK(m, af, cov90Service.ID, cov90Service.PushActions)
			},
			expectRes:       &action.ActionNext,
			expectForwarded: true,
		},
		{
			name:       "connect pushes the confbridge join",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConnect,
			option:     map[string]any{},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				cbID := uuid.FromStringOrNil("c9b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c10")
				callID := uuid.FromStringOrNil("c9b1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c11")
				m.req.EXPECT().CallV1ConfbridgeCreate(ctx, af.CustomerID, af.ID, cmconfbridge.ReferenceTypeCall, af.ReferenceID, cmconfbridge.TypeConnect).Return(&cmconfbridge.Confbridge{ID: cbID}, nil)
				m.req.EXPECT().FlowV1FlowCreate(ctx, af.CustomerID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), uuid.Nil, false).Return(&cov90Flow, nil)
				m.req.EXPECT().CallV1CallsCreate(ctx, cov90Flow.CustomerID, cov90Flow.ID, af.ReferenceID, gomock.Any(), gomock.Any(), false, true, gomock.Any(), nil, nil).Return([]*cmcall.Call{{Identity: commonidentity.Identity{ID: callID}}}, nil, nil)
				m.util.EXPECT().UUIDCreate().Return(cov90ActionID)
				m.stack.EXPECT().PushStackByActions(af.StackMap, uuid.Nil, gomock.Any(), af.CurrentStackID, af.CurrentAction.ID).
					Return(&stack.Stack{ID: cov90StackID, Actions: []action.Action{{ID: cov90ActionID}}}, nil)
				m.db.EXPECT().ActiveflowUpdate(ctx, af.ID, gomock.Any()).Return(nil)
			},
			expectRes:       &action.ActionNext,
			expectForwarded: true,
		},
		{
			name:       "goto with no loop left moves on",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeGoto,
			option:     map[string]any{"target_id": cov90ActionID.String(), "loop_count": 0},
			setup:      func(m *cov90Mocks, af *activeflow.Activeflow) {},
			expectRes:  &action.ActionNext,
		},
		{
			name:       "message_send sends the message",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeMessageSend,
			option:     map[string]any{"text": "synthetic text"},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().MessageV1MessageSend(ctx, uuid.Nil, af.CustomerID, gomock.Any(), gomock.Any(), "synthetic text").Return(&mmmessage.Message{}, nil)
			},
			expectRes: &action.ActionNext,
		},
		{
			name:       "fetch pushes the fetched actions",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeFetch,
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.action.EXPECT().ActionFetchGet(ctx, &af.CurrentAction, af.ID, af.ReferenceID).Return(cov90Service.PushActions, nil)
				expectCov90PushOK(m, af, uuid.Nil, cov90Service.PushActions)
			},
			expectRes:       &action.ActionNext,
			expectForwarded: true,
		},
		{
			name:       "fetch_flow pushes the actions of the flow",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeFetchFlow,
			option:     map[string]any{"flow_id": cov90ActionID.String()},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				f := cov90Flow
				f.CustomerID = af.CustomerID
				f.Actions = cov90Service.PushActions
				m.req.EXPECT().FlowV1FlowGet(ctx, cov90ActionID).Return(&f, nil)
				expectCov90PushOK(m, af, uuid.Nil, cov90Service.PushActions)
			},
			expectRes:       &action.ActionNext,
			expectForwarded: true,
		},
		{
			name:       "queue_join pushes the actions and marks the queuecall waiting",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeQueueJoin,
			option:     map[string]any{"queue_id": cov90ActionID.String()},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().QueueV1ServiceTypeQueuecallStart(ctx, cov90ActionID, af.ID, gomock.Any(), af.ReferenceID).Return(cov90Service, nil)
				expectCov90PushOK(m, af, cov90Service.ID, cov90Service.PushActions)
				m.req.EXPECT().QueueV1QueuecallUpdateStatusWaiting(ctx, cov90Service.ID).Return(nil, nil)
			},
			expectRes:       &action.ActionNext,
			expectForwarded: true,
		},
		{
			name:       "stop pushes the finish action",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeStop,
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				expectCov90PushOK(m, af, uuid.Nil, []action.Action{action.ActionFinish})
			},
			expectRes:       &action.ActionNext,
			expectForwarded: true,
		},
		{
			name:       "case_create with an unreadable option does not stop the flow",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeCaseCreate,
			option:     map[string]any{"sync": "yes"},
			setup:      func(m *cov90Mocks, af *activeflow.Activeflow) {},
			expectRes:  &action.ActionNext,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newCov90Handler(mc)
			af := newCov90Activeflow(tt.refType, tt.actionType, tt.option)
			if !h.verifyActionType(af) {
				t.Fatalf("The action type is not allowed for the reference type. action: %s, reference: %s", tt.actionType, tt.refType)
			}
			tt.setup(m, af)

			res, err := h.executeAction(context.Background(), af)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if res != tt.expectRes {
				t.Errorf("Wrong result.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}

			if tt.expectForwarded {
				if af.ForwardStackID != cov90StackID || af.ForwardActionID != cov90ActionID {
					t.Errorf("The activeflow must point to the pushed actions. stack: %s, action: %s", af.ForwardStackID, af.ForwardActionID)
				}
			} else if af.ForwardStackID != uuid.Nil || af.ForwardActionID != uuid.Nil {
				t.Errorf("The activeflow must not move. stack: %s, action: %s", af.ForwardStackID, af.ForwardActionID)
			}
		})
	}
}

// expectCov90NextAction registers one successful round of updateNextAction for the given next action.
func expectCov90NextAction(m *cov90Mocks, af *activeflow.Activeflow, next *action.Action) {
	m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), af.ID).Return(af, nil)
	m.stack.EXPECT().GetNextAction(gomock.Any(), gomock.Any(), gomock.Any(), true).Return(stack.IDMain, next)
	m.variable.EXPECT().Get(gomock.Any(), af.ID).Return(&variable.Variable{}, nil)
	m.variable.EXPECT().SubstituteOption(gomock.Any(), next.Option, gomock.Any())
	m.db.EXPECT().ActiveflowUpdate(gomock.Any(), af.ID, gomock.Any()).Return(nil)
	m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), af.CustomerID, activeflow.EventTypeActiveflowUpdated, gomock.Any())
	m.db.EXPECT().ActiveflowReleaseLock(gomock.Any(), af.ID).Return(nil)
}

func Test_ExecuteNextAction_flowTermination_coverage90(t *testing.T) {
	ctx := context.Background()

	t.Run("the finish action stops the activeflow and is returned", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)
		af.StackMap = map[uuid.UUID]*stack.Stack{}

		next := action.ActionFinish
		expectCov90NextAction(m, af, &next)
		// updateCurrentAction reads twice and stop reads once and then the updated one.
		m.db.EXPECT().ActiveflowGet(gomock.Any(), af.ID).Return(af, nil).Times(4)
		// the stop of the activeflow
		m.db.EXPECT().ActiveflowUpdate(gomock.Any(), af.ID, map[activeflow.Field]any{activeflow.FieldStatus: activeflow.StatusEnded}).Return(nil)
		m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), af.CustomerID, activeflow.EventTypeActiveflowUpdated, gomock.Any())

		res, err := h.ExecuteNextAction(ctx, af.ID, af.CurrentAction.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res != &action.ActionFinish {
			t.Errorf("The finish action must be returned. got: %v", res)
		}
	})

	t.Run("an action failure stops the activeflow and reports the failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)
		af.StackMap = map[uuid.UUID]*stack.Stack{}

		// the option does not fit the goto option, so the handler fails.
		next := &action.Action{ID: cov90ActionID, Type: action.TypeGoto, Option: map[string]any{"target_id": 123}}
		expectCov90NextAction(m, af, next)
		// the two reads of updateCurrentAction succeed and the read of the stop fails.
		gomock.InOrder(
			m.db.EXPECT().ActiveflowGet(gomock.Any(), af.ID).Return(af, nil).Times(2),
			m.db.EXPECT().ActiveflowGet(gomock.Any(), af.ID).Return(nil, errDepInjected),
		)

		res, err := h.ExecuteNextAction(ctx, af.ID, af.CurrentAction.ID)
		if err == nil || !strings.Contains(err.Error(), "could not execute the active action") {
			t.Fatalf("expected the execute failure, got: %v", err)
		}
		if res != nil {
			t.Errorf("no action is expected with the failure. got: %v", res)
		}
	})

	t.Run("an action that moves on is followed by the next action", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)
		af.StackMap = map[uuid.UUID]*stack.Stack{}

		first := &action.Action{ID: cov90StackID, Type: action.TypeEmpty}
		second := &action.Action{ID: cov90ActionID, Type: action.TypeAnswer}
		gomock.InOrder(
			m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), af.ID).Return(af, nil),
			m.stack.EXPECT().GetNextAction(gomock.Any(), gomock.Any(), gomock.Any(), true).Return(stack.IDMain, first),
		)
		m.variable.EXPECT().Get(gomock.Any(), af.ID).Return(&variable.Variable{}, nil).Times(2)
		m.variable.EXPECT().SubstituteOption(gomock.Any(), gomock.Any(), gomock.Any()).Times(2)
		m.db.EXPECT().ActiveflowGet(gomock.Any(), af.ID).Return(af, nil).Times(4)
		m.db.EXPECT().ActiveflowUpdate(gomock.Any(), af.ID, gomock.Any()).Return(nil).Times(2)
		m.notify.EXPECT().PublishWebhookEvent(gomock.Any(), af.CustomerID, activeflow.EventTypeActiveflowUpdated, gomock.Any()).Times(2)
		m.db.EXPECT().ActiveflowReleaseLock(gomock.Any(), af.ID).Return(nil).Times(2)
		// the second round
		m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), af.ID).Return(af, nil)
		m.stack.EXPECT().GetNextAction(gomock.Any(), gomock.Any(), first.ID, true).Return(stack.IDMain, second)

		res, err := h.ExecuteNextAction(ctx, af.ID, af.CurrentAction.ID)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if res == nil || res.ID != second.ID {
			t.Errorf("The second action must be returned. got: %v", res)
		}
	})
}

func Test_Execute_failure_coverage90(t *testing.T) {
	ctx := context.Background()

	t.Run("Execute reports the failure of the next action", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		id := uuid.FromStringOrNil("c9a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e01")

		m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), id).Return(nil, errDepInjected)
		// the stop of the activeflow can not read it either.
		m.db.EXPECT().ActiveflowGet(gomock.Any(), id).Return(nil, errDepInjected)

		err := h.Execute(ctx, id)
		if err == nil || !strings.Contains(err.Error(), "could not execute the next action") {
			t.Errorf("expected the next action failure, got: %v", err)
		}
	})

	t.Run("ExecuteContinue reports the activeflow read failure", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		id := uuid.FromStringOrNil("c9a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e01")

		m.db.EXPECT().ActiveflowGet(gomock.Any(), id).Return(nil, errDepInjected)

		err := h.ExecuteContinue(ctx, id, cov90ActionID)
		if err == nil || !strings.Contains(err.Error(), "could not get activeflow info") {
			t.Errorf("expected the read failure, got: %v", err)
		}
	})

	t.Run("ExecuteContinue refuses an activeflow that is not blocked", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeAnswer, nil)

		// no other dependency is registered, so the flow must not be executed.
		m.db.EXPECT().ActiveflowGet(gomock.Any(), af.ID).Return(af, nil)

		err := h.ExecuteContinue(ctx, af.ID, af.CurrentAction.ID)
		if err == nil || !strings.Contains(err.Error(), "the activeflow is not blocked") {
			t.Errorf("expected the not blocked failure, got: %v", err)
		}
	})

	t.Run("ExecuteContinue reports the failure of the next action", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeBlock, nil)

		// the read of ExecuteContinue succeeds and the read of the stop fails.
		gomock.InOrder(
			m.db.EXPECT().ActiveflowGet(gomock.Any(), af.ID).Return(af, nil),
			m.db.EXPECT().ActiveflowGetWithLock(gomock.Any(), af.ID).Return(nil, errDepInjected),
			m.db.EXPECT().ActiveflowGet(gomock.Any(), af.ID).Return(nil, errDepInjected),
		)

		err := h.ExecuteContinue(ctx, af.ID, af.CurrentAction.ID)
		if err == nil || !strings.Contains(err.Error(), "could not execute the next action") {
			t.Errorf("expected the next action failure, got: %v", err)
		}
	})
}

package activeflowhandler

import (
	"context"
	"fmt"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/stack"
	"monorepo/bin-flow-manager/pkg/actionhandler"
	"monorepo/bin-flow-manager/pkg/dbhandler"
	"monorepo/bin-flow-manager/pkg/stackmaphandler"
	"monorepo/bin-flow-manager/pkg/variablehandler"
)

// newExecuteErrorTestHandler returns a handler whose dependencies are strict mocks.
// Any unexpected call to a dependency fails the test.
func newExecuteErrorTestHandler(mc *gomock.Controller) (*activeflowHandler, *requesthandler.MockRequestHandler, *actionhandler.MockActionHandler, *stackmaphandler.MockStackmapHandler) {
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockAction := actionhandler.NewMockActionHandler(mc)
	mockStack := stackmaphandler.NewMockStackmapHandler(mc)

	h := &activeflowHandler{
		utilHandler:     utilhandler.NewMockUtilHandler(mc),
		db:              dbhandler.NewMockDBHandler(mc),
		reqHandler:      mockReq,
		notifyHandler:   notifyhandler.NewMockNotifyHandler(mc),
		actionHandler:   mockAction,
		variableHandler: variablehandler.NewMockVariableHandler(mc),
		stackmapHandler: mockStack,
	}

	return h, mockReq, mockAction, mockStack
}

func newExecuteErrorTestActiveflow(refType activeflow.ReferenceType, actionType action.Type, option map[string]any) *activeflow.Activeflow {
	return &activeflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("a3a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e01"),
			CustomerID: uuid.FromStringOrNil("a3a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e02"),
		},
		ReferenceType:  refType,
		CurrentStackID: stack.IDMain,
		CurrentAction: action.Action{
			ID:     uuid.FromStringOrNil("a3a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e03"),
			Type:   actionType,
			Option: option,
		},
	}
}

// Test_executeAction_invalidOption verifies that an option which can not be decoded
// into the action's option struct is reported as an error for the actions
// that propagate handler errors.
func Test_executeAction_invalidOption(t *testing.T) {
	// the values below have a JSON type that does not match the option field.
	tests := []struct {
		name string

		refType    activeflow.ReferenceType
		actionType action.Type
		option     map[string]any
	}{
		{"ai_summary", activeflow.ReferenceTypeConversation, action.TypeAISummary, map[string]any{"on_end_flow_id": 123}},
		{"ai_talk", activeflow.ReferenceTypeCall, action.TypeAITalk, map[string]any{"ai_id": 123}},
		{"ai_task", activeflow.ReferenceTypeConversation, action.TypeAITask, map[string]any{"ai_id": 123}},
		{"branch", activeflow.ReferenceTypeCall, action.TypeBranch, map[string]any{"default_target_id": 123}},
		{"call", activeflow.ReferenceTypeCall, action.TypeCall, map[string]any{"flow_id": 123}},
		{"condition_call_digits", activeflow.ReferenceTypeCall, action.TypeConditionCallDigits, map[string]any{"length": "three"}},
		{"condition_call_status", activeflow.ReferenceTypeCall, action.TypeConditionCallStatus, map[string]any{"false_target_id": 123}},
		{"condition_datetime", activeflow.ReferenceTypeCall, action.TypeConditionDatetime, map[string]any{"minute": "ten"}},
		{"condition_variable", activeflow.ReferenceTypeCall, action.TypeConditionVariable, map[string]any{"variable": 123}},
		{"conference_join", activeflow.ReferenceTypeCall, action.TypeConferenceJoin, map[string]any{"conference_id": 123}},
		{"connect", activeflow.ReferenceTypeCall, action.TypeConnect, map[string]any{"source": "not-an-address"}},
		{"goto", activeflow.ReferenceTypeCall, action.TypeGoto, map[string]any{"target_id": 123}},
		{"message_send", activeflow.ReferenceTypeCall, action.TypeMessageSend, map[string]any{"text": 123}},
		{"queue_join", activeflow.ReferenceTypeCall, action.TypeQueueJoin, map[string]any{"queue_id": 123}},
		{"fetch_flow", activeflow.ReferenceTypeCall, action.TypeFetchFlow, map[string]any{"flow_id": 123}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _, _ := newExecuteErrorTestHandler(mc)
			af := newExecuteErrorTestActiveflow(tt.refType, tt.actionType, tt.option)

			// the action must pass the reference type verification, otherwise the
			// action is skipped and the error path is never reached.
			if !h.verifyActionType(af) {
				t.Fatalf("The action type is not allowed for the reference type. action: %s, reference: %s", tt.actionType, tt.refType)
			}

			res, err := h.executeAction(context.Background(), af)
			if err == nil {
				t.Errorf("Wrong match. expect: error, got: nil")
			}
			if res != nil {
				t.Errorf("Wrong match. expect: nil result, got: %v", res)
			}
		})
	}
}

// Test_executeAction_swallowedError verifies that the actions which are allowed to fail
// do not stop the activeflow and move on to the next action.
func Test_executeAction_swallowedError(t *testing.T) {
	tests := []struct {
		name string

		refType    activeflow.ReferenceType
		actionType action.Type
		option     map[string]any
	}{
		{"conversation_send", activeflow.ReferenceTypeConversation, action.TypeConversationSend, map[string]any{"conversation_id": 123}},
		{"email_send", activeflow.ReferenceTypeConversation, action.TypeEmailSend, map[string]any{"destinations": 123}},
		{"transcribe_recording", activeflow.ReferenceTypeCall, action.TypeTranscribeRecording, map[string]any{"language": 123}},
		{"transcribe_start", activeflow.ReferenceTypeCall, action.TypeTranscribeStart, map[string]any{"language": 123}},
		{"variable_set", activeflow.ReferenceTypeCall, action.TypeVariableSet, map[string]any{"key": 123}},
		{"webhook_send", activeflow.ReferenceTypeCall, action.TypeWebhookSend, map[string]any{"sync": "yes"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _, _ := newExecuteErrorTestHandler(mc)
			af := newExecuteErrorTestActiveflow(tt.refType, tt.actionType, tt.option)

			if !h.verifyActionType(af) {
				t.Fatalf("The action type is not allowed for the reference type. action: %s, reference: %s", tt.actionType, tt.refType)
			}

			res, err := h.executeAction(context.Background(), af)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res != &action.ActionNext {
				t.Errorf("Wrong match. expect: next action, got: %v", res)
			}
		})
	}
}

// Test_executeAction_dependencyError verifies the error propagation of the actions
// whose failure comes from a dependency instead of the option.
func Test_executeAction_dependencyError(t *testing.T) {
	t.Run("fetch", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, mockAction, _ := newExecuteErrorTestHandler(mc)
		af := newExecuteErrorTestActiveflow(activeflow.ReferenceTypeCall, action.TypeFetch, nil)

		mockAction.EXPECT().ActionFetchGet(gomock.Any(), &af.CurrentAction, af.ID, af.ReferenceID).Return(nil, fmt.Errorf("fetch failed"))

		res, err := h.executeAction(context.Background(), af)
		if err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil result, got: %v", res)
		}
	})

	t.Run("fetch_flow", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, mockReq, _, _ := newExecuteErrorTestHandler(mc)
		flowID := uuid.FromStringOrNil("a3a1ba1c-1b5f-4b3e-9d2a-0f6a3c9d1e04")
		af := newExecuteErrorTestActiveflow(activeflow.ReferenceTypeCall, action.TypeFetchFlow, map[string]any{"flow_id": flowID.String()})

		mockReq.EXPECT().FlowV1FlowGet(gomock.Any(), flowID).Return(nil, fmt.Errorf("flow get failed"))

		res, err := h.executeAction(context.Background(), af)
		if err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil result, got: %v", res)
		}
	})

	t.Run("stop", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, _, _, mockStack := newExecuteErrorTestHandler(mc)
		af := newExecuteErrorTestActiveflow(activeflow.ReferenceTypeCall, action.TypeStop, nil)

		mockStack.EXPECT().PushStackByActions(af.StackMap, uuid.Nil, []action.Action{action.ActionFinish}, af.CurrentStackID, af.CurrentAction.ID).Return(nil, fmt.Errorf("push failed"))

		res, err := h.executeAction(context.Background(), af)
		if err == nil {
			t.Errorf("Wrong match. expect: error, got: nil")
		}
		if res != nil {
			t.Errorf("Wrong match. expect: nil result, got: %v", res)
		}
	})
}

// Test_executeAction_skippedAndEmpty verifies the actions which are not executed by the flow-manager itself.
func Test_executeAction_skippedAndEmpty(t *testing.T) {
	tests := []struct {
		name string

		refType    activeflow.ReferenceType
		actionType action.Type
	}{
		{
			// the action needs a realtime media, but the reference is a non-realtime one.
			// the action is skipped without touching any dependency.
			name:       "action not allowed for the reference type",
			refType:    activeflow.ReferenceTypeConversation,
			actionType: action.TypeConnect,
		},
		{
			name:       "empty action",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeEmpty,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _, _ := newExecuteErrorTestHandler(mc)
			af := newExecuteErrorTestActiveflow(tt.refType, tt.actionType, nil)

			res, err := h.executeAction(context.Background(), af)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
			if res != &action.ActionNext {
				t.Errorf("Wrong match. expect: next action, got: %v", res)
			}
		})
	}
}

package activeflowhandler

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
)

// Test_actionHandle_unmarshalableOption verifies that the action handlers stop
// before calling any dependency when the option can not be encoded, and keep the
// encoding failure as the cause of the returned error.
func Test_actionHandle_unmarshalableOption(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		call func(h *activeflowHandler, af *activeflow.Activeflow) error
	}{
		{"fetch_flow", func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleFetchFlow(ctx, af) }},
		{"condition_call_digits", func(h *activeflowHandler, af *activeflow.Activeflow) error {
			return h.actionHandleConditionCallDigits(ctx, af)
		}},
		{"condition_call_status", func(h *activeflowHandler, af *activeflow.Activeflow) error {
			return h.actionHandleConditionCallStatus(ctx, af)
		}},
		{"condition_datetime", func(h *activeflowHandler, af *activeflow.Activeflow) error {
			return h.actionHandleConditionDatetime(ctx, af)
		}},
		{"condition_variable", func(h *activeflowHandler, af *activeflow.Activeflow) error {
			return h.actionHandleConditionVariable(ctx, af)
		}},
		{"conference_join", func(h *activeflowHandler, af *activeflow.Activeflow) error {
			return h.actionHandleConferenceJoin(ctx, af)
		}},
		{"connect", func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleConnect(ctx, af) }},
		{"goto", func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleGoto(ctx, af) }},
		{"transcribe_recording", func(h *activeflowHandler, af *activeflow.Activeflow) error {
			return h.actionHandleTranscribeRecording(ctx, af)
		}},
		{"transcribe_start", func(h *activeflowHandler, af *activeflow.Activeflow) error {
			return h.actionHandleTranscribeStart(ctx, af)
		}},
		{"queue_join", func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleQueueJoin(ctx, af) }},
		{"branch", func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleBranch(ctx, af) }},
		{"message_send", func(h *activeflowHandler, af *activeflow.Activeflow) error {
			return h.actionHandleMessageSend(ctx, af)
		}},
		{"call", func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleCall(ctx, af) }},
		{"variable_set", func(h *activeflowHandler, af *activeflow.Activeflow) error {
			return h.actionHandleVariableSet(ctx, af)
		}},
		{"webhook_send", func(h *activeflowHandler, af *activeflow.Activeflow) error {
			return h.actionHandleWebhookSend(ctx, af)
		}},
		{"conversation_send", func(h *activeflowHandler, af *activeflow.Activeflow) error {
			return h.actionHandleConversationSend(ctx, af)
		}},
		{"ai_talk", func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleAITalk(ctx, af) }},
		{"email_send", func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleEmailSend(ctx, af) }},
		{"ai_summary", func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleAISummary(ctx, af) }},
		{"ai_task", func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleAITask(ctx, af) }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, _, _, _ := newExecuteErrorTestHandler(mc)
			af := newExecuteErrorTestActiveflow(activeflow.ReferenceTypeCall, action.TypeAnswer, map[string]any{"bad": make(chan int)})

			err := tt.call(h, af)
			if err == nil || !strings.Contains(err.Error(), "could not marshal the option") {
				t.Fatalf("Wrong match. expect: marshal error, got: %v", err)
			}

			var unsupported *json.UnsupportedTypeError
			if !errors.As(err, &unsupported) {
				t.Errorf("Wrong match. expect: the encoding failure as the cause, got: %v", err)
			}
		})
	}
}

// Test_actionHandleCaseCreate_unmarshalableOption verifies that case_create is
// best effort: an option that can not be encoded is dropped without an error and
// without calling any dependency.
func Test_actionHandleCaseCreate_unmarshalableOption(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	h, _, _, _ := newExecuteErrorTestHandler(mc)
	af := newExecuteErrorTestActiveflow(activeflow.ReferenceTypeCall, action.TypeCaseCreate, map[string]any{"bad": make(chan int)})

	if err := h.actionHandleCaseCreate(context.Background(), af); err != nil {
		t.Errorf("Wrong match. expect: nil, got: %v", err)
	}
}

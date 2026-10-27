package activeflowhandler

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	cmcall "monorepo/bin-call-manager/models/call"
	commonaddress "monorepo/bin-common-handler/models/address"
	cmkase "monorepo/bin-contact-manager/models/kase"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	wmwebhook "monorepo/bin-webhook-manager/models/webhook"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/variable"
)

// waitCov90 waits for the channel, and fails the test when it does not close in time.
func waitCov90(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatalf("The asynchronous call did not finish")
	}
}

func Test_actionHandleCaseCreate_tolerated_coverage90(t *testing.T) {
	ctx := context.Background()

	callAF := func() *activeflow.Activeflow {
		return newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeCaseCreate, map[string]any{"note": "synthetic note"})
	}
	conversationAF := func() *activeflow.Activeflow {
		return newCov90Activeflow(activeflow.ReferenceTypeConversation, action.TypeCaseCreate, map[string]any{"note": "synthetic note"})
	}
	// the peer type is not known to the normalizer, so the raw target is used.
	unknownPeer := commonaddress.Address{Type: commonaddress.Type("unknown_type"), Target: "raw target"}
	self := commonaddress.Address{Type: commonaddress.TypeAgent, Target: "self target"}
	createdCase := &cmkase.Case{ID: uuid.FromStringOrNil("c9d1c0de-1e2f-4a3b-8c4d-5e6f7a8b9c01")}

	tests := []struct {
		name string

		af    func() *activeflow.Activeflow
		setup func(m *cov90Mocks, af *activeflow.Activeflow)
	}{
		{
			name: "the call can not be read",
			af:   callAF,
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallGet(ctx, af.ReferenceID).Return(nil, errDepInjected)
			},
		},
		{
			name: "the conversation can not be read",
			af:   conversationAF,
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().ConversationV1ConversationGet(ctx, af.ReferenceID).Return(nil, errDepInjected)
			},
		},
		{
			name: "the raw target is used when it can not be normalized and the follow up requests fail",
			af:   conversationAF,
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().ConversationV1ConversationGet(ctx, af.ReferenceID).Return(&cvconversation.Conversation{Self: self, Peer: unknownPeer}, nil)
				m.req.EXPECT().FlowV1VariableGet(ctx, af.ID).Return(&variable.Variable{Variables: map[string]string{}}, nil)
				m.req.EXPECT().ContactV1CaseCreate(ctx, af.CustomerID, self, unknownPeer, "conversation_message", "", "", af.ReferenceID.String()).Return(createdCase, nil)
				m.req.EXPECT().FlowV1VariableSetVariable(ctx, af.ID, map[string]string{"contact_case_id": createdCase.ID.String()}).Return(errDepInjected)
				m.req.EXPECT().ContactV1CaseNoteCreate(ctx, af.CustomerID, createdCase.ID, "system", nil, "synthetic note").Return(nil, errDepInjected)
			},
		},
		{
			name: "a call with a source and a destination creates the case for the remote party",
			af:   callAF,
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				c := &cmcall.Call{
					Direction:   cmcall.DirectionOutgoing,
					Source:      self,
					Destination: unknownPeer,
				}
				m.req.EXPECT().CallV1CallGet(ctx, af.ReferenceID).Return(c, nil)
				m.req.EXPECT().FlowV1VariableGet(ctx, af.ID).Return(nil, errDepInjected)
				m.req.EXPECT().ContactV1CaseCreate(ctx, af.CustomerID, self, unknownPeer, "call", "", "", af.ReferenceID.String()).Return(nil, errDepInjected)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newCov90Handler(mc)
			af := tt.af()
			tt.setup(m, af)

			// the action is not critical, so no failure of the case creation is reported.
			if err := h.actionHandleCaseCreate(ctx, af); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test_actionHandleTranscribe_coverage90(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		refType activeflow.ReferenceType
		call    func(h *activeflowHandler, af *activeflow.Activeflow) error
		setup   func(m *cov90Mocks, af *activeflow.Activeflow)

		expectFail bool
	}{
		{
			name:    "transcribe_recording does nothing for a reference that is not a call",
			refType: activeflow.ReferenceTypeConversation,
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleTranscribeRecording(ctx, af)
			},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {},
		},
		{
			name:    "transcribe_recording reports the failure of the transcribe request",
			refType: activeflow.ReferenceTypeCall,
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleTranscribeRecording(ctx, af)
			},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().CallV1CallGet(ctx, af.ReferenceID).Return(&cmcall.Call{RecordingIDs: []uuid.UUID{cov90ActionID}}, nil)
				m.req.EXPECT().TranscribeV1TranscribeStart(ctx, uuid.Nil, af.CustomerID, af.ID, gomock.Any(), gomock.Any(), cov90ActionID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errDepInjected)
			},
			expectFail: true,
		},
		{
			name:    "transcribe_start does nothing for a reference that is not a call",
			refType: activeflow.ReferenceTypeConversation,
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleTranscribeStart(ctx, af)
			},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {},
		},
		{
			name:    "transcribe_start reports the failure of the transcribe request",
			refType: activeflow.ReferenceTypeCall,
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleTranscribeStart(ctx, af)
			},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().TranscribeV1TranscribeStart(ctx, uuid.Nil, af.CustomerID, af.ID, gomock.Any(), gomock.Any(), af.ReferenceID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errDepInjected)
			},
			expectFail: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newCov90Handler(mc)
			af := newCov90Activeflow(tt.refType, action.TypeTranscribeStart, map[string]any{})
			tt.setup(m, af)

			err := tt.call(h, af)
			if tt.expectFail {
				if !errors.Is(err, errDepInjected) {
					t.Errorf("The cause must be kept. got: %v", err)
				}
				return
			}
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func Test_actionHandleWebhookSend_and_ConversationSend_coverage90(t *testing.T) {
	ctx := context.Background()
	message := &cvmessage.Message{}

	t.Run("a failed webhook on the synchronous mode is tolerated", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeWebhookSend, map[string]any{"sync": true, "uri": "https://example.invalid/hook", "method": "POST", "data_type": "application/json", "data": "{}"})
		m.req.EXPECT().WebhookV1WebhookSendToDestination(ctx, af.CustomerID, "https://example.invalid/hook", wmwebhook.MethodType("POST"), wmwebhook.DataType("application/json"), []byte("{}")).Return(errDepInjected)

		if err := h.actionHandleWebhookSend(ctx, af); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
	})

	t.Run("a failed webhook on the asynchronous mode does not reach the caller", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeWebhookSend, map[string]any{"uri": "https://example.invalid/hook", "method": "POST", "data_type": "application/json", "data": "{}"})

		sent := make(chan struct{})
		m.req.EXPECT().WebhookV1WebhookSendToDestination(ctx, af.CustomerID, "https://example.invalid/hook", wmwebhook.MethodType("POST"), wmwebhook.DataType("application/json"), []byte("{}")).DoAndReturn(
			func(_ context.Context, _ uuid.UUID, _ string, _ wmwebhook.MethodType, _ wmwebhook.DataType, _ []byte) error {
				defer close(sent)
				return errDepInjected
			})

		if err := h.actionHandleWebhookSend(ctx, af); err != nil {
			t.Errorf("unexpected error: %v", err)
		}
		waitCov90(t, sent)
	})

	t.Run("a failed conversation message on the synchronous mode is reported", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		h, m := newCov90Handler(mc)
		af := newCov90Activeflow(activeflow.ReferenceTypeConversation, action.TypeConversationSend, map[string]any{"conversation_id": cov90ActionID.String(), "text": "synthetic text", "sync": true})
		m.req.EXPECT().ConversationV1MessageSend(ctx, cov90ActionID, "synthetic text", gomock.Any()).Return(nil, errDepInjected)

		if err := h.actionHandleConversationSend(ctx, af); !errors.Is(err, errDepInjected) {
			t.Errorf("The cause must be kept. got: %v", err)
		}
	})

	for _, tt := range []struct {
		name     string
		response *cvmessage.Message
		err      error
	}{
		{"a conversation message on the asynchronous mode is sent in the background", message, nil},
		{"a failed conversation message on the asynchronous mode does not reach the caller", nil, errDepInjected},
	} {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newCov90Handler(mc)
			af := newCov90Activeflow(activeflow.ReferenceTypeConversation, action.TypeConversationSend, map[string]any{"conversation_id": cov90ActionID.String(), "text": "synthetic text"})

			sent := make(chan struct{})
			m.req.EXPECT().ConversationV1MessageSend(ctx, cov90ActionID, "synthetic text", gomock.Any()).DoAndReturn(
				func(_ context.Context, _ uuid.UUID, _ string, _ any) (*cvmessage.Message, error) {
					defer close(sent)
					return tt.response, tt.err
				})

			if err := h.actionHandleConversationSend(ctx, af); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			waitCov90(t, sent)
		})
	}
}

func Test_actionHandle_serviceStartFailure_coverage90(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name string

		refType    activeflow.ReferenceType
		actionType action.Type
		option     map[string]any
		call       func(h *activeflowHandler, af *activeflow.Activeflow) error
		setup      func(m *cov90Mocks, af *activeflow.Activeflow)

		expectMessage string
	}{
		{
			name:       "ai_talk: the service start fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeAITalk,
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleAITalk(ctx, af) },
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().AIV1ServiceTypeAIcallStart(ctx, gomock.Any(), gomock.Any(), af.ID, gomock.Any(), af.ReferenceID, gomock.Any()).Return(nil, errDepInjected)
			},
			expectMessage: "Could not start the service.",
		},
		{
			name:       "ai_talk: pushing the actions fails",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeAITalk,
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleAITalk(ctx, af) },
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().AIV1ServiceTypeAIcallStart(ctx, gomock.Any(), gomock.Any(), af.ID, gomock.Any(), af.ReferenceID, gomock.Any()).Return(cov90Service, nil)
				m.stack.EXPECT().PushStackByActions(af.StackMap, cov90Service.ID, cov90Service.PushActions, af.CurrentStackID, af.CurrentAction.ID).Return(nil, errDepInjected)
			},
			expectMessage: "Could not push the actions to the stack.",
		},
		{
			name:       "ai_summary: the service start fails",
			refType:    activeflow.ReferenceTypeConversation,
			actionType: action.TypeAISummary,
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleAISummary(ctx, af) },
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().AIV1ServiceTypeSummaryStart(ctx, af.CustomerID, af.ID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(nil, errDepInjected)
			},
			expectMessage: "Could not start the service.",
		},
		{
			name:       "ai_summary: pushing the actions fails",
			refType:    activeflow.ReferenceTypeConversation,
			actionType: action.TypeAISummary,
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleAISummary(ctx, af) },
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().AIV1ServiceTypeSummaryStart(ctx, af.CustomerID, af.ID, gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Return(cov90Service, nil)
				m.stack.EXPECT().PushStackByActions(af.StackMap, cov90Service.ID, cov90Service.PushActions, af.CurrentStackID, af.CurrentAction.ID).Return(nil, errDepInjected)
			},
			expectMessage: "Could not push the actions to the stack.",
		},
		{
			name:       "ai_task: the service start fails",
			refType:    activeflow.ReferenceTypeConversation,
			actionType: action.TypeAITask,
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleAITask(ctx, af) },
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().AIV1ServiceTypeTaskStart(ctx, gomock.Any(), gomock.Any(), af.ID).Return(nil, errDepInjected)
			},
			expectMessage: "Could not start the service.",
		},
		{
			name:       "ai_task: pushing the actions fails",
			refType:    activeflow.ReferenceTypeConversation,
			actionType: action.TypeAITask,
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleAITask(ctx, af) },
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().AIV1ServiceTypeTaskStart(ctx, gomock.Any(), gomock.Any(), af.ID).Return(cov90Service, nil)
				m.stack.EXPECT().PushStackByActions(af.StackMap, cov90Service.ID, cov90Service.PushActions, af.CurrentStackID, af.CurrentAction.ID).Return(nil, errDepInjected)
			},
			expectMessage: "Could not push the actions to the stack.",
		},
		{
			name:       "email_send: the send fails",
			refType:    activeflow.ReferenceTypeConversation,
			actionType: action.TypeEmailSend,
			option:     map[string]any{"subject": "synthetic subject", "content": "synthetic content"},
			call:       func(h *activeflowHandler, af *activeflow.Activeflow) error { return h.actionHandleEmailSend(ctx, af) },
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.req.EXPECT().EmailV1EmailSend(ctx, af.CustomerID, af.ID, gomock.Any(), "synthetic subject", "synthetic content", gomock.Any()).Return(nil, errDepInjected)
			},
			expectMessage: errDepInjected.Error(),
		},
		{
			// the minute 60 never equals a real minute, so the condition never matches
			// no matter what the current time is.
			name:       "condition_datetime: a minute that never matches goes to the false target",
			refType:    activeflow.ReferenceTypeCall,
			actionType: action.TypeConditionDatetime,
			option:     map[string]any{"condition": "==", "hour": -1, "minute": 60, "false_target_id": depFalseTargetID.String()},
			call: func(h *activeflowHandler, af *activeflow.Activeflow) error {
				return h.actionHandleConditionDatetime(ctx, af)
			},
			setup: func(m *cov90Mocks, af *activeflow.Activeflow) {
				m.stack.EXPECT().GetAction(af.StackMap, af.CurrentStackID, depFalseTargetID, false).Return(cov90StackID, nil, errDepInjected)
			},
			expectMessage: "could not find false target action.",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			h, m := newCov90Handler(mc)
			option := tt.option
			if option == nil {
				option = map[string]any{}
			}
			af := newCov90Activeflow(tt.refType, tt.actionType, option)
			tt.setup(m, af)

			err := tt.call(h, af)
			if err == nil || !strings.Contains(err.Error(), tt.expectMessage) {
				t.Fatalf("Wrong error.\nexpect to contain: %s\ngot: %v", tt.expectMessage, err)
			}
			if !errors.Is(err, errDepInjected) {
				t.Errorf("The cause must be kept. got: %v", err)
			}
		})
	}
}

func Test_actionHandleBranch_resetFailure_coverage90(t *testing.T) {
	ctx := context.Background()

	mc := gomock.NewController(t)
	defer mc.Finish()

	h, m := newCov90Handler(mc)
	af := newCov90Activeflow(activeflow.ReferenceTypeCall, action.TypeBranch, map[string]any{"default_target_id": cov90ActionID.String()})

	m.variable.EXPECT().Get(ctx, af.ID).Return(&variable.Variable{Variables: map[string]string{}}, nil)
	// the reset of the variable fails but the flow keeps going to the target.
	m.variable.EXPECT().SetVariable(ctx, af.ID, gomock.Any()).Return(errDepInjected)
	m.stack.EXPECT().GetAction(af.StackMap, af.CurrentStackID, cov90ActionID, false).Return(cov90StackID, &action.Action{ID: cov90ActionID}, nil)
	m.db.EXPECT().ActiveflowUpdate(ctx, af.ID, gomock.Any()).Return(nil)

	if err := h.actionHandleBranch(ctx, af); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if af.ForwardStackID != cov90StackID || af.ForwardActionID != cov90ActionID {
		t.Errorf("The activeflow must point to the target. stack: %s, action: %s", af.ForwardStackID, af.ForwardActionID)
	}
}

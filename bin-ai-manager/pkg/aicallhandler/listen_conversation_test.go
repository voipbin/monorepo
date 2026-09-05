package aicallhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"

	"monorepo/bin-ai-manager/internal/config"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	commonidentity "monorepo/bin-common-handler/models/identity"
	kmkase "monorepo/bin-contact-manager/models/kase"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

var (
	lcConversationID = uuid.FromStringOrNil("44440000-0000-4000-8000-000000000001")
)

// listeningConversationAIcall is an AIcall that passes every conversation-kind
// RunListenTurn precondition. ltAIcallID/ltCustomerID/ltCaseID come from
// listen_trigger_test.go.
func listeningConversationAIcall() *aicall.AIcall {
	return &aicall.AIcall{
		Identity:       commonidentity.Identity{ID: ltAIcallID, CustomerID: ltCustomerID},
		AssistanceType: aicall.AssistanceTypeAI,
		AssistanceID:   ltAIID,
		ReferenceType:  aicall.ReferenceTypeContactCase,
		ReferenceID:    ltCaseID,
		Status:         aicall.StatusProgressing,
		PipecatcallID:  uuid.FromStringOrNil("44440000-0000-4000-8000-0000000000aa"),
		Metadata: map[string]any{
			aicall.MetaKeyListenConversationID: lcConversationID.String(),
		},
	}
}

func openConversationCase() *kmkase.Case {
	return &kmkase.Case{
		ID:            ltCaseID,
		CustomerID:    ltCustomerID,
		ReferenceType: kmkase.ReferenceTypeConversationMessage,
		ReferenceID:   lcConversationID.String(),
		Status:        kmkase.StatusOpen,
	}
}

func Test_listenKindOf(t *testing.T) {
	tests := []struct {
		name   string
		c      *aicall.AIcall
		expect listenKind
		label  string
	}{
		{"nil metadata", &aicall.AIcall{}, listenKindNone, "unknown"},
		{"transcribe pointer", listeningAIcall(), listenKindCall, "call"},
		{"conversation pointer", listeningConversationAIcall(), listenKindConversation, "conversation"},
		{"malformed conversation pointer", &aicall.AIcall{Metadata: map[string]any{aicall.MetaKeyListenConversationID: "nope"}}, listenKindNone, "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listenKindOf(tt.c)
			if got != tt.expect {
				t.Errorf("kind mismatch. expected: %q, got: %q", tt.expect, got)
			}
			if got.label() != tt.label {
				t.Errorf("label mismatch. expected: %q, got: %q", tt.label, got.label())
			}
		})
	}
}

// Test_RunListenTurn_Conversation covers the conversation-only gates that sit
// between the predicate and the turn counter (design 2026-09-05 §5.5.1, §5.5.2).
// Not parallel: metric deltas.
func Test_RunListenTurn_Conversation(t *testing.T) {
	tests := []struct {
		name              string
		callKind          bool
		conversationFlag  bool
		pendingLen        int64
		pendingLenErr     error
		responseCase      *kmkase.Case
		responseCaseErr   error
		expectStop        bool
		expectCaseGet     bool
		expectCounter     bool
		expectResultLabel string
	}{
		{
			name:              "conversation sub-flag off stops the session without a Case RPC",
			conversationFlag:  false,
			expectStop:        true,
			expectResultLabel: "skipped_disabled",
		},
		{
			name:              "empty pending buffer short-circuits before the Case RPC and the counter",
			conversationFlag:  true,
			pendingLen:        0,
			expectResultLabel: "skipped_empty",
		},
		{
			name:              "LLEN error is tolerated and the turn proceeds to the Case check",
			conversationFlag:  true,
			pendingLenErr:     fmt.Errorf("redis down"),
			responseCase:      openConversationCase(),
			expectCaseGet:     true,
			expectCounter:     true,
			expectResultLabel: "skipped_empty", // ListenPendingPopAll returns nothing in this harness, so the turn ends as skipped_empty after the counter
		},
		{
			name:              "closed Case stops listening",
			conversationFlag:  true,
			pendingLen:        2,
			responseCase:      &kmkase.Case{ID: ltCaseID, CustomerID: ltCustomerID, ReferenceType: kmkase.ReferenceTypeConversationMessage, ReferenceID: lcConversationID.String(), Status: kmkase.StatusClosed},
			expectCaseGet:     true,
			expectStop:        true,
			expectResultLabel: "skipped_case_closed",
		},
		{
			name:              "Case RPC failure is metered as failed and nothing is popped or counted",
			conversationFlag:  true,
			pendingLen:        2,
			responseCaseErr:   fmt.Errorf("rpc timeout"),
			expectCaseGet:     true,
			expectResultLabel: "failed",
		},
		{
			// Design §7 item 4: a CALL-kind AIcall in the same table must be
			// untouched by the conversation gates -- no LLEN, no Case RPC, and
			// the conversation sub-flag being OFF must not stop it.
			name:              "call kind ignores the conversation sub-flag and never calls LLEN or the Case RPC",
			callKind:          true,
			conversationFlag:  false,
			expectCounter:     true,
			expectResultLabel: "skipped_empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			config.SetListenDefaultsForTest()
			config.SetAIcallListenEnabledForTest(true)
			config.SetAIcallListenConversationEnabledForTest(tt.conversationFlag)

			mc := gomock.NewController(t)
			defer mc.Finish()
			m := newListenTurnHarness(mc)
			ctx := context.Background()

			c := listeningConversationAIcall()
			kindLabel := "conversation"
			if tt.callKind {
				c = listeningAIcall()
				kindLabel = "call"
			}
			m.db.EXPECT().AIcallGet(ctx, ltAIcallID).Return(c, nil)

			if tt.conversationFlag && !tt.callKind {
				m.cache.EXPECT().ListenPendingLen(ctx, ltAIcallID).Return(tt.pendingLen, tt.pendingLenErr)
			} else {
				m.cache.EXPECT().ListenPendingLen(gomock.Any(), gomock.Any()).Times(0)
			}
			if tt.expectCaseGet {
				m.req.EXPECT().ContactV1CaseGet(ctx, ltCustomerID, ltCaseID).Return(tt.responseCase, tt.responseCaseErr)
			} else {
				m.req.EXPECT().ContactV1CaseGet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}
			if tt.expectCounter {
				m.cache.EXPECT().ListenTurnCountIncr(ctx, ltAIcallID, gomock.Any()).Return(int64(1), nil)
				m.cache.EXPECT().ListenPendingPopAll(ctx, ltAIcallID).Return([]string{}, nil)
			} else {
				m.cache.EXPECT().ListenTurnCountIncr(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
				m.cache.EXPECT().ListenPendingPopAll(gomock.Any(), gomock.Any()).Times(0)
			}
			if tt.expectStop {
				// stopListening -> clearListenState for the conversation kind:
				// SREM the resolver, clear the keys, strip the metadata key.
				m.cache.EXPECT().ListenConversationAIcallIDRemove(ctx, lcConversationID, ltAIcallID).Return(nil)
				m.cache.EXPECT().ListenStateClear(ctx, ltAIcallID).Return(nil)
				m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).DoAndReturn(
					func(_ context.Context, _ uuid.UUID, fields map[aicall.Field]any) error {
						meta, _ := fields[aicall.FieldMetadata].(map[string]any)
						if _, still := meta[aicall.MetaKeyListenConversationID]; still {
							t.Errorf("clearListenState must strip listen_conversation_id")
						}
						return nil
					})
			} else {
				m.cache.EXPECT().ListenStateClear(gomock.Any(), gomock.Any()).Times(0)
			}

			got := metricDelta(t, kindLabel, tt.expectResultLabel, func() { m.h.RunListenTurn(ctx, ltAIcallID) })
			if got != 1 {
				t.Errorf("result %q must be metered exactly once. got: %v", tt.expectResultLabel, got)
			}
		})
	}
}

func Test_buildListenTurnMessages_ConversationKind(t *testing.T) {
	config.SetListenDefaultsForTest()

	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newListenTurnHarness(mc)
	ctx := context.Background()

	c := listeningConversationAIcall()
	m.msg.EXPECT().List(ctx, uint64(30), "", gomock.Any()).Return([]*message.Message{}, nil)

	res, err := m.h.buildListenTurnMessages(ctx, c, []string{"[CUSTOMER] hi"}, []string{"[CUSTOMER] hi"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var sawConversationPrompt, sawCallPrompt bool
	for _, row := range res {
		content, _ := row["content"].(string)
		if content == ListenTurnConversationSystemPrompt {
			sawConversationPrompt = true
		}
		if content == ListenTurnSystemPrompt {
			sawCallPrompt = true
		}
	}
	if !sawConversationPrompt || sawCallPrompt {
		t.Errorf("conversation kind must use ListenTurnConversationSystemPrompt only. conversation: %v, call: %v", sawConversationPrompt, sawCallPrompt)
	}

	last, _ := res[len(res)-1]["content"].(string)
	if !strings.HasPrefix(last, "Conversation so far:\n") {
		t.Errorf("conversation transcript block must start with the conversation header. got: %q", last)
	}
}

// Test_ProcessTerminate_ConversationListenGate pins process.go's widened gate:
// a conversation listener (ListenCallID == Nil, metadata pointer set) must
// still have its state cleared on terminate.
func Test_clearListenState_ConversationKind(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()
	m := newListenTurnHarness(mc)
	ctx := context.Background()

	c := listeningConversationAIcall()

	rem := m.cache.EXPECT().ListenConversationAIcallIDRemove(ctx, lcConversationID, ltAIcallID).Return(nil)
	clear := m.cache.EXPECT().ListenStateClear(ctx, ltAIcallID).Return(nil)
	update := m.db.EXPECT().AIcallUpdateNoTouchTMUpdate(ctx, ltAIcallID, gomock.Any()).Return(nil)
	gomock.InOrder(rem, clear, update)
	m.cache.EXPECT().ListenAIcallIDRemove(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
	m.req.EXPECT().TranscribeV1TranscribeGet(gomock.Any(), gomock.Any()).Times(0)

	m.h.stopListening(ctx, c)
}

func Test_listenTerminateNeedsStop(t *testing.T) {
	tests := []struct {
		name   string
		c      *aicall.AIcall
		expect bool
	}{
		{"nil", nil, false},
		{"task reference is never listening", &aicall.AIcall{ReferenceType: aicall.ReferenceTypeTask, ListenCallID: ltCallID}, false},
		{"contact_case with no listen state", &aicall.AIcall{ReferenceType: aicall.ReferenceTypeContactCase}, false},
		{"contact_case with listen_call_id column", &aicall.AIcall{ReferenceType: aicall.ReferenceTypeContactCase, ListenCallID: ltCallID}, true},
		{"contact_case with transcribe pointer", listeningAIcall(), true},
		{"contact_case with conversation pointer", listeningConversationAIcall(), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := listenTerminateNeedsStop(tt.c); got != tt.expect {
				t.Errorf("mismatch. expected: %v, got: %v", tt.expect, got)
			}
		})
	}
}

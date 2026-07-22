package aicallhandler

import (
	"context"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"

	cminteraction "monorepo/bin-contact-manager/models/interaction"
	kmkase "monorepo/bin-contact-manager/models/kase"
	cvmessage "monorepo/bin-conversation-manager/models/message"
)

func testAIcallForCase(customerID, caseID uuid.UUID) *aicall.AIcall {
	return &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("6a1f2c10-c001-11f0-9000-000000000001"),
			CustomerID: customerID,
		},
		ReferenceType: aicall.ReferenceTypeContactCase,
		ReferenceID:   caseID,
	}
}

// Test_toolHandleGetContactInteractions covers design VOIP-1234 §4: Case
// fetch -> contact_id-preferred / peer-fallback interaction list, ownership
// masking, and empty-result-is-success (never a failure).
func Test_toolHandleGetContactInteractions(t *testing.T) {
	customerID := uuid.FromStringOrNil("6a1f2c10-c001-11f0-9000-000000000002")
	caseID := uuid.FromStringOrNil("6a1f2c10-c001-11f0-9000-000000000003")
	contactID := uuid.FromStringOrNil("6a1f2c10-c001-11f0-9000-000000000004")
	toolCallID := "6a1f2c10-c001-11f0-9000-000000000005"

	tc := &message.ToolCall{
		ID:   toolCallID,
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameGetContactInteractions,
			Arguments: `{}`,
		},
	}

	tests := []struct {
		name string

		responseCase        *kmkase.Case
		responseCaseErr     error
		responseInteraction []*cminteraction.Interaction
		responseListErr     error

		expectContactFilter bool // true: filter by contact_id; false: filter by peer
		expectResult        string
		expectMessageEmpty  bool
	}{
		{
			name: "contact_id set -> filter by contact",
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
				Peer: commonaddress.Address{Type: "tel", Target: "+15551500001"},
			},
			responseInteraction: []*cminteraction.Interaction{
				{
					Direction:     "incoming",
					Peer: commonaddress.Address{Type: "tel", Target: "+15551500001"},
					ReferenceType: "conversation_message",
					ReferenceID:   uuid.FromStringOrNil("6a1f2c10-c001-11f0-9000-000000000010"),
					TMInteraction: timePtr(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)),
				},
			},
			expectContactFilter: true,
			expectResult:        "success",
		},
		{
			name: "no contact_id -> filter by peer",
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Peer: commonaddress.Address{Type: "tel", Target: "+15551500002"},
			},
			responseInteraction: []*cminteraction.Interaction{
				{
					Direction:     "outgoing",
					Peer: commonaddress.Address{Type: "tel", Target: "+15551500002"},
					ReferenceType: "call",
					ReferenceID:   uuid.FromStringOrNil("6a1f2c10-c001-11f0-9000-000000000011"),
				},
			},
			expectContactFilter: false,
			expectResult:        "success",
		},
		{
			name: "empty interaction list -> success, not failed",
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				Peer: commonaddress.Address{Type: "tel", Target: "+15551500003"},
			},
			responseInteraction: []*cminteraction.Interaction{},
			expectContactFilter: false,
			expectResult:        "success",
			expectMessageEmpty:  true,
		},
		{
			name:            "case not found -> masked, not failed",
			responseCaseErr: requesthandler.ErrNotFound,
			expectResult:    "success",
		},
		{
			name: "cross-customer case -> masked",
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: uuid.FromStringOrNil("6a1f2c10-c001-11f0-9000-0000000000ff"), // different customer
			},
			expectResult: "success",
		},
		{
			name:            "case RPC transient failure -> honest failure",
			responseCaseErr: errTest,
			expectResult:    "failed",
		},
		{
			name: "Contact soft-deleted (typed CONTACT_NOT_FOUND from InteractionList) -> success, not failed",
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
				Peer: commonaddress.Address{Type: "tel", Target: "+15551500004"},
			},
			responseListErr:     cerrors.NotFound(commonoutline.ServiceNameContactManager, "CONTACT_NOT_FOUND", "The contact was not found."),
			expectContactFilter: true,
			expectResult:        "success",
			expectMessageEmpty:  true,
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &aicallHandler{reqHandler: mockReq}
			ctx := context.Background()
			c := testAIcallForCase(customerID, caseID)

			mockReq.EXPECT().ContactV1CaseGet(ctx, customerID, caseID).Return(tt.responseCase, tt.responseCaseErr)

			if tt.responseCaseErr == nil && tt.responseCase != nil && tt.responseCase.CustomerID == customerID {
				if tt.expectContactFilter {
					mockReq.EXPECT().ContactV1InteractionList(
						ctx, customerID, uint64(insightDefaultListLimit), "", "", "", contactID, uuid.Nil, time.Time{},
					).Return(tt.responseInteraction, "", tt.responseListErr)
				} else {
					mockReq.EXPECT().ContactV1InteractionList(
						ctx, customerID, uint64(insightDefaultListLimit), "", string(tt.responseCase.Peer.Type), tt.responseCase.Peer.Target, uuid.Nil, uuid.Nil, time.Time{},
					).Return(tt.responseInteraction, "", tt.responseListErr)
				}
			}

			res := h.toolHandleGetContactInteractions(ctx, c, tc)

			if res.Result != tt.expectResult {
				t.Fatalf("Result = %q, want %q (message: %s)", res.Result, tt.expectResult, res.Message)
			}
			if tt.expectResult == "success" && tt.responseCaseErr == nil && tt.responseCase != nil && tt.responseCase.CustomerID == customerID {
				if tt.expectMessageEmpty && res.Message != "no interactions found" {
					t.Errorf("expected empty-result message, got: %s", res.Message)
				}
			}
			if tt.responseCaseErr != nil && tt.responseCaseErr != requesthandler.ErrNotFound {
				if res.Message != "resource lookup failed" {
					t.Errorf("expected honest failure message, got: %s", res.Message)
				}
			}
		})
	}
}

// Test_toolHandleGetConversationContent covers design VOIP-1234 §5: explicit
// reference_id (LLM must discover it via get_contact_interactions first),
// ownership masking on the resolved message (IDOR defense), and a FIXED
// 2-RPC resolution (MessageGet + one MessageList filtered by conversation_id)
// regardless of message/thread count -- this is the regression guard against
// the rejected N+1 per-message-fetch draft.
func Test_toolHandleGetConversationContent(t *testing.T) {
	customerID := uuid.FromStringOrNil("6b1f2c10-c001-11f0-9000-000000000002")
	caseID := uuid.FromStringOrNil("6b1f2c10-c001-11f0-9000-000000000003")
	refID := uuid.FromStringOrNil("6b1f2c10-c001-11f0-9000-000000000010")
	conversationID := uuid.FromStringOrNil("6b1f2c10-c001-11f0-9000-000000000020")
	toolCallID := "6b1f2c10-c001-11f0-9000-000000000005"

	c := testAIcallForCase(customerID, caseID)

	t.Run("missing reference_id -> failed, no RPC calls", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq}
		ctx := context.Background()

		tc := &message.ToolCall{
			ID:   toolCallID,
			Type: message.ToolTypeFunction,
			Function: message.FunctionCall{
				Name:      message.FunctionCallNameGetConversationContent,
				Arguments: `{}`,
			},
		}
		res := h.toolHandleGetConversationContent(ctx, c, tc)
		if res.Result != "failed" {
			t.Fatalf("Result = %q, want failed", res.Result)
		}
	})

	t.Run("happy path: fixed 2 RPCs, one MessageGet + one MessageList filtered by conversation_id", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq}
		ctx := context.Background()

		tc := &message.ToolCall{
			ID:   toolCallID,
			Type: message.ToolTypeFunction,
			Function: message.FunctionCall{
				Name:      message.FunctionCallNameGetConversationContent,
				Arguments: `{"reference_id":"` + refID.String() + `"}`,
			},
		}

		resolvedMsg := &cvmessage.Message{
			Identity:       commonidentity.Identity{ID: refID, CustomerID: customerID},
			ConversationID: conversationID,
		}
		mockReq.EXPECT().ConversationV1MessageGet(ctx, refID).Return(resolvedMsg, nil).Times(1)

		threadMsgs := []cvmessage.Message{
			{Identity: commonidentity.Identity{CustomerID: customerID}, Direction: "incoming", Text: "hello", TMCreate: timePtr(time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC))},
			{Identity: commonidentity.Identity{CustomerID: customerID}, Direction: "outgoing", Text: "hi there", TMCreate: timePtr(time.Date(2026, 7, 1, 0, 1, 0, 0, time.UTC))},
		}
		mockReq.EXPECT().ConversationV1MessageList(
			ctx, "", uint64(insightDefaultListLimit), map[cvmessage.Field]any{cvmessage.FieldConversationID: conversationID.String()},
		).Return(threadMsgs, nil).Times(1)

		res := h.toolHandleGetConversationContent(ctx, c, tc)
		if res.Result != "success" {
			t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
		}
	})

	t.Run("message not found -> masked, not failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq}
		ctx := context.Background()

		tc := &message.ToolCall{
			ID:   toolCallID,
			Type: message.ToolTypeFunction,
			Function: message.FunctionCall{
				Name:      message.FunctionCallNameGetConversationContent,
				Arguments: `{"reference_id":"` + refID.String() + `"}`,
			},
		}
		mockReq.EXPECT().ConversationV1MessageGet(ctx, refID).Return(nil, requesthandler.ErrNotFound)
		// no MessageList call expected -- masking happens before the second RPC.

		res := h.toolHandleGetConversationContent(ctx, c, tc)
		if res.Result != "success" || res.Message != msgResourceNotFound {
			t.Fatalf("expected masked not-found, got Result=%q Message=%q", res.Result, res.Message)
		}
	})

	t.Run("cross-customer message -> masked (IDOR defense)", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq}
		ctx := context.Background()

		tc := &message.ToolCall{
			ID:   toolCallID,
			Type: message.ToolTypeFunction,
			Function: message.FunctionCall{
				Name:      message.FunctionCallNameGetConversationContent,
				Arguments: `{"reference_id":"` + refID.String() + `"}`,
			},
		}
		foreignMsg := &cvmessage.Message{
			Identity:       commonidentity.Identity{ID: refID, CustomerID: uuid.FromStringOrNil("6b1f2c10-c001-11f0-9000-0000000000ff")},
			ConversationID: conversationID,
		}
		mockReq.EXPECT().ConversationV1MessageGet(ctx, refID).Return(foreignMsg, nil)
		// no MessageList call expected -- masking happens before the second RPC.

		res := h.toolHandleGetConversationContent(ctx, c, tc)
		if res.Result != "success" || res.Message != msgResourceNotFound {
			t.Fatalf("expected masked not-found, got Result=%q Message=%q", res.Result, res.Message)
		}
	})

	t.Run("empty thread -> success, not failed", func(t *testing.T) {
		mc := gomock.NewController(t)
		defer mc.Finish()
		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq}
		ctx := context.Background()

		tc := &message.ToolCall{
			ID:   toolCallID,
			Type: message.ToolTypeFunction,
			Function: message.FunctionCall{
				Name:      message.FunctionCallNameGetConversationContent,
				Arguments: `{"reference_id":"` + refID.String() + `"}`,
			},
		}
		resolvedMsg := &cvmessage.Message{
			Identity:       commonidentity.Identity{ID: refID, CustomerID: customerID},
			ConversationID: conversationID,
		}
		mockReq.EXPECT().ConversationV1MessageGet(ctx, refID).Return(resolvedMsg, nil)
		mockReq.EXPECT().ConversationV1MessageList(
			ctx, "", uint64(insightDefaultListLimit), map[cvmessage.Field]any{cvmessage.FieldConversationID: conversationID.String()},
		).Return([]cvmessage.Message{}, nil)

		res := h.toolHandleGetConversationContent(ctx, c, tc)
		if res.Result != "success" || res.Message != "no messages found" {
			t.Fatalf("expected empty-result success, got Result=%q Message=%q", res.Result, res.Message)
		}
	})
}

func timePtr(t time.Time) *time.Time {
	return &t
}

// Test_isNotFoundErr covers both error shapes this codebase's downstream
// managers use for "not found" (Round-2 review finding, VOIP-1234 PR
// #1100): the legacy requesthandler.ErrNotFound sentinel AND a typed
// *cerrors.VoipbinError with Status == StatusNotFound. A caller that checks
// only one shape silently misclassifies the other as an honest failure.
func Test_isNotFoundErr(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "legacy sentinel",
			err:  requesthandler.ErrNotFound,
			want: true,
		},
		{
			name: "typed VoipbinError NotFound",
			err:  cerrors.NotFound(commonoutline.ServiceNameContactManager, "CONTACT_NOT_FOUND", "The contact was not found."),
			want: true,
		},
		{
			name: "typed VoipbinError, different status -> not a not-found",
			err:  cerrors.PermissionDenied(commonoutline.ServiceNameContactManager, "X", "x"),
			want: false,
		},
		{
			name: "unrelated error",
			err:  errTest,
			want: false,
		},
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if got := isNotFoundErr(tt.err); got != tt.want {
				t.Errorf("isNotFoundErr(%v) = %v, want %v", tt.err, got, tt.want)
			}
		})
	}
}

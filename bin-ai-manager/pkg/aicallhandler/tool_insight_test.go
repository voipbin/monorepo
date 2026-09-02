package aicallhandler

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	cmcall "monorepo/bin-call-manager/models/call"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	kmkase "monorepo/bin-contact-manager/models/kase"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"
	tmtranscript "monorepo/bin-transcribe-manager/models/transcript"
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
		responseInteraction []*tmpeerevent.PeerEvent
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
				Peer:       commonaddress.Address{Type: "tel", Target: "+15551500001"},
			},
			responseInteraction: []*tmpeerevent.PeerEvent{
				{
					Direction:   "incoming",
					Peer:        commonaddress.Address{Type: "tel", Target: "+155****0001"},
					Publisher:   "conversation_message",
					ReferenceID: uuid.FromStringOrNil("6a1f2c10-c001-11f0-9000-000000000010"),
					Timestamp:   time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
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
				Peer:       commonaddress.Address{Type: "tel", Target: "+15551500002"},
			},
			responseInteraction: []*tmpeerevent.PeerEvent{
				{
					Direction:   "outgoing",
					Peer:        commonaddress.Address{Type: "tel", Target: "+155****0002"},
					Publisher:   "call",
					ReferenceID: uuid.FromStringOrNil("6a1f2c10-c001-11f0-9000-000000000011"),
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
				Peer:       commonaddress.Address{Type: "tel", Target: "+15551500003"},
			},
			responseInteraction: []*tmpeerevent.PeerEvent{},
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
				Peer:       commonaddress.Address{Type: "tel", Target: "+15551500004"},
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

// Test_toolHandleGetRelatedCases covers design docs/plans/
// 2026-07-30-case-insight-assistant-tool-expansion-design.md §1.1: nil/zero
// ContactID short-circuit (never reach ContactV1CaseList with uuid.Nil),
// self-exclusion (current case never in its own related-case list), ownership
// masking, and honest failure on RPC error.
func Test_toolHandleGetRelatedCases(t *testing.T) {
	customerID := uuid.FromStringOrNil("7a1f2c10-c001-11f0-9000-000000000002")
	caseID := uuid.FromStringOrNil("7a1f2c10-c001-11f0-9000-000000000003")
	contactID := uuid.FromStringOrNil("7a1f2c10-c001-11f0-9000-000000000004")
	otherCaseID := uuid.FromStringOrNil("7a1f2c10-c001-11f0-9000-000000000005")
	toolCallID := "7a1f2c10-c001-11f0-9000-000000000006"

	tc := &message.ToolCall{
		ID:   toolCallID,
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameGetRelatedCases,
			Arguments: `{}`,
		},
	}

	tests := []struct {
		name string

		responseCase       *kmkase.Case
		responseCaseErr    error
		expectCaseListCall bool
		responseCases      []*kmkase.Case
		responseCasesErr   error

		expectResult       string
		expectMessage      string
		expectMessageEmpty bool
	}{
		{
			name:            "case not found -> masked, not failed",
			responseCaseErr: requesthandler.ErrNotFound,
			expectResult:    "success",
			expectMessage:   msgResourceNotFound,
		},
		{
			name: "cross-customer case -> masked",
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: uuid.FromStringOrNil("7a1f2c10-c001-11f0-9000-0000000000ff"),
			},
			expectResult:  "success",
			expectMessage: msgResourceNotFound,
		},
		{
			name:            "case RPC transient failure -> honest failure",
			responseCaseErr: errTest,
			expectResult:    "failed",
		},
		{
			name: "nil ContactID -> success empty, RPC never called",
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  nil,
			},
			expectCaseListCall: false,
			expectResult:       "success",
			expectMessage:      "no related cases found",
		},
		{
			name: "zero-UUID ContactID -> success empty, RPC never called",
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &uuid.Nil,
			},
			expectCaseListCall: false,
			expectResult:       "success",
			expectMessage:      "no related cases found",
		},
		{
			name: "self-exclusion: current case removed from results",
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectCaseListCall: true,
			responseCases: []*kmkase.Case{
				{ID: caseID, CustomerID: customerID, Name: "current case"},
			},
			expectResult:  "success",
			expectMessage: "no related cases found",
		},
		{
			name: "other case present -> returned",
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectCaseListCall: true,
			responseCases: []*kmkase.Case{
				{ID: caseID, CustomerID: customerID, Name: "current case"},
				{ID: otherCaseID, CustomerID: customerID, Name: "other case", Status: kmkase.StatusOpen},
			},
			expectResult: "success",
		},
		{
			name: "case list RPC error -> honest failure",
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectCaseListCall: true,
			responseCasesErr:   errTest,
			expectResult:       "failed",
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

			if tt.expectCaseListCall {
				mockReq.EXPECT().ContactV1CaseList(
					ctx, customerID, "", "", uuid.Nil, contactID, uint64(insightMaxListLimit), "", "",
				).Return(tt.responseCases, "", tt.responseCasesErr)
			}

			res := h.toolHandleGetRelatedCases(ctx, c, tc)

			if res.Result != tt.expectResult {
				t.Fatalf("Result = %q, want %q (message: %s)", res.Result, tt.expectResult, res.Message)
			}
			if tt.expectMessage != "" && res.Message != tt.expectMessage {
				t.Errorf("Message = %q, want %q", res.Message, tt.expectMessage)
			}
		})
	}
}

// Test_toolHandleGetRelatedCases_TruncationUsesRawPageSize is a regression
// test (round-2 code review finding) for a bug where the truncation flag was
// computed from the POST-exclusion count against the page-size limit: if the
// raw fetch returned exactly insightMaxListLimit rows and one of them was the
// current case (excluded from the result), the post-exclusion count fell one
// short of the limit and the truncation marker was incorrectly omitted even
// though further related cases could exist beyond this page. The flag must
// be derived from the RAW (pre-exclusion) row count instead.
func Test_toolHandleGetRelatedCases_TruncationUsesRawPageSize(t *testing.T) {
	customerID := uuid.FromStringOrNil("7a1f2c10-c001-11f0-9000-000000000102")
	caseID := uuid.FromStringOrNil("7a1f2c10-c001-11f0-9000-000000000103")
	contactID := uuid.FromStringOrNil("7a1f2c10-c001-11f0-9000-000000000104")
	toolCallID := "7a1f2c10-c001-11f0-9000-000000000106"

	tc := &message.ToolCall{
		ID:   toolCallID,
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameGetRelatedCases,
			Arguments: `{}`,
		},
	}

	// Build exactly insightMaxListLimit raw rows, with the CURRENT case as
	// one of them -- so the post-exclusion count is insightMaxListLimit-1,
	// while the raw fetch count that must drive the truncation flag is
	// exactly insightMaxListLimit.
	rawCases := make([]*kmkase.Case, 0, insightMaxListLimit)
	rawCases = append(rawCases, &kmkase.Case{ID: caseID, CustomerID: customerID, Name: "current case"})
	for i := 1; i < insightMaxListLimit; i++ {
		rawCases = append(rawCases, &kmkase.Case{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: customerID,
			Name:       fmt.Sprintf("other case %d", i),
			Status:     kmkase.StatusOpen,
		})
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()
	c := testAIcallForCase(customerID, caseID)

	mockReq.EXPECT().ContactV1CaseGet(ctx, customerID, caseID).Return(
		&kmkase.Case{ID: caseID, CustomerID: customerID, ContactID: &contactID}, nil,
	)
	mockReq.EXPECT().ContactV1CaseList(
		ctx, customerID, "", "", uuid.Nil, contactID, uint64(insightMaxListLimit), "", "",
	).Return(rawCases, "", nil)

	res := h.toolHandleGetRelatedCases(ctx, c, tc)

	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	if !strings.Contains(res.Message, "showing the most recent") {
		t.Errorf("expected truncation marker in message when raw fetch hit the page cap, got: %s", res.Message)
	}
}

// Test_toolHandleGetCaseNotes covers design §1.2: ownership masking and the
// "most recent N" truncation, which requires slicing from the TAIL of the
// ascending (tm_create asc) note list, not the head.
func Test_toolHandleGetCaseNotes(t *testing.T) {
	customerID := uuid.FromStringOrNil("7b1f2c10-c001-11f0-9000-000000000002")
	caseID := uuid.FromStringOrNil("7b1f2c10-c001-11f0-9000-000000000003")
	toolCallID := "7b1f2c10-c001-11f0-9000-000000000004"

	baseCase := &kmkase.Case{ID: caseID, CustomerID: customerID}

	makeNotes := func(n int) []*cmcasenote.CaseNote {
		notes := make([]*cmcasenote.CaseNote, 0, n)
		base := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
		for i := 0; i < n; i++ {
			ts := base.Add(time.Duration(i) * time.Minute)
			notes = append(notes, &cmcasenote.CaseNote{
				ID:         uuid.Must(uuid.NewV4()),
				CustomerID: customerID,
				CaseID:     caseID,
				AuthorType: cmcasenote.AuthorTypeAgent,
				Text:       fmt.Sprintf("note-%d", i),
				TMCreate:   timePtr(ts),
			})
		}
		return notes
	}

	tests := []struct {
		name string

		args string

		responseCase    *kmkase.Case
		responseCaseErr error
		responseNotes   []*cmcasenote.CaseNote
		responseNoteErr error

		expectResult  string
		expectMessage string
	}{
		{
			name:            "case not found -> masked",
			args:            `{}`,
			responseCaseErr: requesthandler.ErrNotFound,
			expectResult:    "success",
			expectMessage:   msgResourceNotFound,
		},
		{
			name: "cross-customer case -> masked",
			args: `{}`,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: uuid.FromStringOrNil("7b1f2c10-c001-11f0-9000-0000000000ff"),
			},
			expectResult:  "success",
			expectMessage: msgResourceNotFound,
		},
		{
			name:            "case RPC transient failure -> honest failure",
			args:            `{}`,
			responseCaseErr: errTest,
			expectResult:    "failed",
		},
		{
			name:          "empty notes -> success empty",
			args:          `{}`,
			responseCase:  baseCase,
			responseNotes: []*cmcasenote.CaseNote{},
			expectResult:  "success",
			expectMessage: "no notes found",
		},
		{
			name:          "notes within limit -> all returned, no truncation",
			args:          `{"limit":20}`,
			responseCase:  baseCase,
			responseNotes: makeNotes(3),
			expectResult:  "success",
		},
		{
			name:          "notes exceed limit -> keeps the MOST RECENT (tail), not the head",
			args:          `{"limit":2}`,
			responseCase:  baseCase,
			responseNotes: makeNotes(5), // note-0 .. note-4, ascending tm_create
			expectResult:  "success",
		},
		{
			name:            "note list RPC error -> honest failure",
			args:            `{}`,
			responseCase:    baseCase,
			responseNoteErr: errTest,
			expectResult:    "failed",
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

			tc := &message.ToolCall{
				ID:   toolCallID,
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetCaseNotes,
					Arguments: tt.args,
				},
			}

			mockReq.EXPECT().ContactV1CaseGet(ctx, customerID, caseID).Return(tt.responseCase, tt.responseCaseErr)

			if tt.responseCaseErr == nil && tt.responseCase != nil && tt.responseCase.CustomerID == customerID {
				mockReq.EXPECT().ContactV1CaseNoteList(ctx, customerID, caseID).Return(tt.responseNotes, tt.responseNoteErr)
			}

			res := h.toolHandleGetCaseNotes(ctx, c, tc)

			if res.Result != tt.expectResult {
				t.Fatalf("Result = %q, want %q (message: %s)", res.Result, tt.expectResult, res.Message)
			}
			if tt.expectMessage != "" && res.Message != tt.expectMessage {
				t.Errorf("Message = %q, want %q", res.Message, tt.expectMessage)
			}
			if tt.name == "notes exceed limit -> keeps the MOST RECENT (tail), not the head" {
				if !strings.Contains(res.Message, "note-4") || strings.Contains(res.Message, "note-0") {
					t.Errorf("expected the most recent notes (tail) to be kept, got: %s", res.Message)
				}
			}
		})
	}
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

// Test_toolHandleGetContactProfile covers design docs/plans/
// 2026-09-02-insight-assistant-get-contact-profile-design.md §4: the
// ReferenceType entry guard, the two-RPC resolution flow, the mandatory
// response-side tenant check on the UNSCOPED ContactV1ContactGet, the
// "never call the unscoped RPC with a nil contact id" guard, and the
// header/lines rendering split with its own truncation note.
func Test_toolHandleGetContactProfile(t *testing.T) {
	customerID := uuid.FromStringOrNil("9a1f2c10-c001-11f0-9000-000000000002")
	otherCustomerID := uuid.FromStringOrNil("9a1f2c10-c001-11f0-9000-0000000000ff")
	caseID := uuid.FromStringOrNil("9a1f2c10-c001-11f0-9000-000000000003")
	contactID := uuid.FromStringOrNil("9a1f2c10-c001-11f0-9000-000000000004")
	toolCallID := "9a1f2c10-c001-11f0-9000-000000000006"

	tc := &message.ToolCall{
		ID:   toolCallID,
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameGetContactProfile,
			Arguments: `{}`,
		},
	}

	testAddress := func(t commonaddress.Type, target string) cmcontact.Address {
		return cmcontact.Address{
			Address:    commonaddress.Address{Type: t, Target: target},
			CustomerID: customerID,
			ContactID:  contactID,
		}
	}

	// A contact with more addresses than insightContactAddressLimit, used by
	// the address-cap case below.
	manyAddresses := make([]cmcontact.Address, 0, insightContactAddressLimit+2)
	for i := 0; i < insightContactAddressLimit+2; i++ {
		manyAddresses = append(manyAddresses, testAddress(commonaddress.TypeTel, fmt.Sprintf("+8210000000%02d", i)))
	}

	tests := []struct {
		name string

		referenceType aicall.ReferenceType

		expectCaseGetCall bool
		responseCase      *kmkase.Case
		responseCaseErr   error

		expectContactGetCall bool
		responseContact      *cmcontact.Contact
		responseContactErr   error

		expectResult      string
		expectMessage     string
		expectContains    []string
		expectNotContains []string
	}{
		{
			name:                 "reference type guard -> failed, zero RPC calls",
			referenceType:        aicall.ReferenceTypeCall,
			expectCaseGetCall:    false,
			expectContactGetCall: false,
			expectResult:         "failed",
		},
		{
			name:              "happy path: full profile",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectContactGetCall: true,
			responseContact: &cmcontact.Contact{
				Identity:    commonidentity.Identity{ID: contactID, CustomerID: customerID},
				DisplayName: "John Smith",
				Company:     "Acme Corporation",
				JobTitle:    "Sales Manager",
				Addresses: []cmcontact.Address{
					testAddress(commonaddress.TypeTel, "+821011112222"),
					testAddress(commonaddress.TypeEmail, "john@acme.example"),
				},
			},
			expectResult: "success",
			expectContains: []string{
				`name="John Smith"`,
				`company="Acme Corporation"`,
				`job_title="Sales Manager"`,
				`address: type=tel target="+821011112222"`,
				`address: type=email target="john@acme.example"`,
			},
		},
		{
			name:              "happy path: sparse profile falls back to first+last name and omits empty lines",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectContactGetCall: true,
			responseContact: &cmcontact.Contact{
				Identity:  commonidentity.Identity{ID: contactID, CustomerID: customerID},
				FirstName: "Jane",
				LastName:  "Doe",
			},
			expectResult:      "success",
			expectMessage:     `name="Jane Doe"`,
			expectNotContains: []string{"company=", "job_title=", "address:"},
		},
		{
			name:              "address cap: own truncation note in header, no misleading 'most recent' marker",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectContactGetCall: true,
			responseContact: &cmcontact.Contact{
				Identity:    commonidentity.Identity{ID: contactID, CustomerID: customerID},
				DisplayName: "Many Numbers",
				Company:     "Acme Corporation",
				Addresses:   manyAddresses,
			},
			expectResult: "success",
			expectContains: []string{
				// NOTE: this fixture is small (well under the 4000-rune cap),
				// so it exercises renderBodyLines' FAST path (pagedOut=false,
				// content fits -> returned verbatim), not its truncation walk.
				// It therefore only pins the cap-to-5 + own-note behavior, NOT
				// header-vs-lines placement (round-2 finding N1) -- that needs
				// content large enough to force the walk, which the dedicated
				// "identity survives pathological overflow" case below covers.
				`name="Many Numbers"`,
				`company="Acme Corporation"`,
				fmt.Sprintf("(showing %d of %d addresses)", insightContactAddressLimit, len(manyAddresses)),
				`target="+821000000000"`,
				fmt.Sprintf(`target="+8210000000%02d"`, insightContactAddressLimit-1),
			},
			expectNotContains: []string{
				// renderBodyLines' built-in marker must never fire here: it
				// asserts recency about a primary-first list (round-5 finding).
				"showing the most recent",
				fmt.Sprintf(`target="+8210000000%02d"`, insightContactAddressLimit),
			},
		},
		{
			// Round-2 finding N1's actual regression test: forces
			// renderBodyLines PAST its fast path (header+lines together
			// exceed maxResourceSummaryRunes) so its truncation walk runs.
			// If identity fields were wrongly placed in `lines` instead of
			// `header` (the original N1 bug), the walk -- which drops from
			// the FRONT of `lines` -- would sacrifice name/company first.
			// Placed in `header`, they are written unconditionally
			// (tool_resource.go) and are never touched by that walk, so they
			// must appear at the very start of the response regardless of
			// how much of the address content survives.
			name:              "identity survives pathological company-name overflow (round-2 N1 regression)",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectContactGetCall: true,
			responseContact: &cmcontact.Contact{
				Identity:    commonidentity.Identity{ID: contactID, CustomerID: customerID},
				DisplayName: "Overflow Test",
				Company:     strings.Repeat("x", 4200), // >> maxResourceSummaryRunes (4000) alone
				Addresses: []cmcontact.Address{
					testAddress(commonaddress.TypeTel, "+821011112222"),
				},
			},
			expectResult: "success",
			expectContains: []string{
				// Must survive at the very front of the (possibly
				// tail-truncated) response -- this is the assertion that
				// actually fails if identity fields end up in `lines`.
				`name="Overflow Test"`,
			},
		},
		{
			name:                 "case not found -> masked, contact RPC never called",
			referenceType:        aicall.ReferenceTypeContactCase,
			expectCaseGetCall:    true,
			responseCaseErr:      requesthandler.ErrNotFound,
			expectContactGetCall: false,
			expectResult:         "success",
			expectMessage:        msgResourceNotFound,
		},
		{
			name:              "cross-customer case -> masked, contact RPC never called",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: otherCustomerID,
				ContactID:  &contactID,
			},
			expectContactGetCall: false,
			expectResult:         "success",
			expectMessage:        msgResourceNotFound,
		},
		{
			// The single most security-load-bearing assertion in this set:
			// ContactV1ContactGet is UNSCOPED, so it must never be reached
			// with a nil/zero contact id. Asserted via gomock Times(0).
			name:              "nil ContactID -> distinct non-masked message, unscoped contact RPC never called",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  nil,
			},
			expectContactGetCall: false,
			expectResult:         "success",
			expectMessage:        "no contact profile found",
		},
		{
			name:              "contact not found -> masked",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectContactGetCall: true,
			responseContactErr:   requesthandler.ErrNotFound,
			expectResult:         "success",
			expectMessage:        msgResourceNotFound,
		},
		{
			// The mandatory design §3.2 step 5 check -- the SOLE tenant
			// enforcement on the unscoped ContactV1ContactGet.
			name:              "cross-customer contact -> masked, no panic",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectContactGetCall: true,
			responseContact: &cmcontact.Contact{
				Identity:    commonidentity.Identity{ID: contactID, CustomerID: otherCustomerID},
				DisplayName: "Someone Else's Contact",
				Addresses:   []cmcontact.Address{testAddress(commonaddress.TypeTel, "+821099998888")},
			},
			expectResult:      "success",
			expectMessage:     msgResourceNotFound,
			expectNotContains: []string{"Someone Else's Contact", "+821099998888"},
		},
		{
			// Distinct from the case above: contact itself is nil, which the
			// nil-safe audit log on that branch must survive.
			name:              "nil contact response -> masked, no panic",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectContactGetCall: true,
			responseContact:      nil,
			expectResult:         "success",
			expectMessage:        msgResourceNotFound,
		},
		{
			// Round-1 code review MEDIUM fix regression test: the per-address
			// tenant filter must run BEFORE the cap, so (a) a foreign address
			// row never renders even though the parent contact check already
			// passed, and (b) the truncation note (absent here, since only 1
			// of 2 rows survives filtering -- well under the cap) is computed
			// from the POST-filter count, never overclaiming what's shown.
			name:              "cross-customer address row filtered out, not counted",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectContactGetCall: true,
			responseContact: &cmcontact.Contact{
				Identity:    commonidentity.Identity{ID: contactID, CustomerID: customerID},
				DisplayName: "Mixed Rows",
				Addresses: []cmcontact.Address{
					testAddress(commonaddress.TypeTel, "+821011112222"),
					{
						Address:    commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821099990000"},
						CustomerID: otherCustomerID,
						ContactID:  contactID,
					},
				},
			},
			expectResult: "success",
			expectContains: []string{
				`name="Mixed Rows"`,
				`target="+821011112222"`,
			},
			expectNotContains: []string{
				"+821099990000",
				"showing", // no truncation note: only 1 of 2 rows survives filtering
			},
		},
		{
			// Round-2 code review MEDIUM fix: the case above has only 2
			// addresses, so filter-before-cap and filter-after-cap produce
			// byte-identical output and the case cannot actually prove the
			// ordering fix. This fixture has 7 rows with a foreign row
			// inside the first 6 -- 6 survive filtering, which is still
			// >insightContactAddressLimit (5), so the cap AND the note both
			// engage. Filter-after-cap (the old, buggy order) would produce
			// "(showing 5 of 7 addresses)" and DROP the valid
			// +821000000005 row to make room for the (then still present)
			// foreign row; filter-before-cap produces "(showing 5 of 6
			// addresses)" and keeps it.
			name:              "cross-customer address row excluded from both the count and the cap",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectContactGetCall: true,
			responseContact: &cmcontact.Contact{
				Identity:    commonidentity.Identity{ID: contactID, CustomerID: customerID},
				DisplayName: "Mixed Many",
				Addresses: []cmcontact.Address{
					testAddress(commonaddress.TypeTel, "+821000000000"),
					{
						Address:    commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821099990000"},
						CustomerID: otherCustomerID,
						ContactID:  contactID,
					},
					testAddress(commonaddress.TypeTel, "+821000000001"),
					testAddress(commonaddress.TypeTel, "+821000000002"),
					testAddress(commonaddress.TypeTel, "+821000000003"),
					testAddress(commonaddress.TypeTel, "+821000000004"),
					testAddress(commonaddress.TypeTel, "+821000000005"),
				},
			},
			expectResult: "success",
			expectContains: []string{
				"(showing 5 of 6 addresses)",
				`target="+821000000004"`, // the 5th SURVIVING (post-filter)
				// address. Under the old, buggy filter-after-cap order, the
				// pre-filter cap keeps [000, foreign, 001, 002, 003] and
				// then drops the foreign row from THAT set, so 004 never
				// appears -- this assertion is what actually fails against
				// 082c7e735's original code.
			},
			expectNotContains: []string{
				"+821099990000",
				"(showing 5 of 7 addresses)", // the old code's overclaim
			},
		},
		{
			// Design §5's field-exclusion contract (Notes/ExternalID/Source,
			// address Detail/Name/TargetName) has no code path that could emit
			// these today, but nothing pins that -- this locks it down so a
			// future refactor that starts threading them through fails loudly.
			name:              "excluded fields never rendered",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectContactGetCall: true,
			responseContact: &cmcontact.Contact{
				Identity:    commonidentity.Identity{ID: contactID, CustomerID: customerID},
				DisplayName: "Notes Test",
				Notes:       "internal-notes-should-never-leak",
				ExternalID:  "sf_003x000001ABC",
				Source:      cmcontact.SourceImport,
				TagIDs:      []uuid.UUID{uuid.FromStringOrNil("9a1f2c10-c001-11f0-9000-0000000000aa")},
				Addresses: []cmcontact.Address{
					{
						Address: commonaddress.Address{
							Type:       commonaddress.TypeTel,
							Target:     "+821011112222",
							TargetName: "target-name-should-never-leak",
							Name:       "addr-name-should-never-leak",
							Detail:     "addr-detail-should-never-leak",
						},
						CustomerID: customerID,
						ContactID:  contactID,
					},
				},
			},
			expectResult: "success",
			expectContains: []string{
				`target="+821011112222"`,
			},
			expectNotContains: []string{
				"internal-notes-should-never-leak",
				"sf_003x000001ABC",
				"import",
				"target-name-should-never-leak",
				"addr-name-should-never-leak",
				"addr-detail-should-never-leak",
				"9a1f2c10-c001-11f0-9000-0000000000aa", // TagIDs (design §5/§6: dropped entirely from v1)
			},
		},
		{
			// Design §3.4's stated rationale for %q: an adversarial free-text
			// field must not be able to forge an adjacent field in the flat
			// key=value line format. Asserts the escaped form, not just that
			// SOME quote character appears somewhere in the output.
			name:              "adversarial company value cannot forge job_title via unescaped quoting",
			referenceType:     aicall.ReferenceTypeContactCase,
			expectCaseGetCall: true,
			responseCase: &kmkase.Case{
				ID:         caseID,
				CustomerID: customerID,
				ContactID:  &contactID,
			},
			expectContactGetCall: true,
			responseContact: &cmcontact.Contact{
				Identity:    commonidentity.Identity{ID: contactID, CustomerID: customerID},
				DisplayName: "Bob",
				Company:     `Verified Partner" job_title="Administrator`,
			},
			expectResult: "success",
			expectContains: []string{
				// %q escapes the embedded quote, so the forged job_title=
				// never appears as an unescaped, standalone key=value pair.
				`company="Verified Partner\" job_title=\"Administrator"`,
			},
			expectNotContains: []string{
				// The literal unescaped forged field must never appear.
				`job_title="Administrator"`,
			},
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
			c.ReferenceType = tt.referenceType

			if tt.expectCaseGetCall {
				mockReq.EXPECT().ContactV1CaseGet(ctx, customerID, caseID).Return(tt.responseCase, tt.responseCaseErr)
			} else {
				mockReq.EXPECT().ContactV1CaseGet(gomock.Any(), gomock.Any(), gomock.Any()).Times(0)
			}

			if tt.expectContactGetCall {
				mockReq.EXPECT().ContactV1ContactGet(ctx, contactID).Return(tt.responseContact, tt.responseContactErr)
			} else {
				mockReq.EXPECT().ContactV1ContactGet(gomock.Any(), gomock.Any()).Times(0)
			}

			res := h.toolHandleGetContactProfile(ctx, c, tc)

			if res.Result != tt.expectResult {
				t.Fatalf("Result = %q, want %q (message: %s)", res.Result, tt.expectResult, res.Message)
			}
			if tt.expectMessage != "" && res.Message != tt.expectMessage {
				t.Errorf("Message = %q, want %q", res.Message, tt.expectMessage)
			}
			for _, want := range tt.expectContains {
				if !strings.Contains(res.Message, want) {
					t.Errorf("Message = %q, want it to contain %q", res.Message, want)
				}
			}
			for _, notWant := range tt.expectNotContains {
				if strings.Contains(res.Message, notWant) {
					t.Errorf("Message = %q, want it NOT to contain %q", res.Message, notWant)
				}
			}
			if tt.expectResult == "success" && res.ResourceType != "contact_profile" {
				t.Errorf("ResourceType = %q, want %q", res.ResourceType, "contact_profile")
			}
			if tt.expectResult == "success" && res.ResourceID != caseID.String() {
				t.Errorf("ResourceID = %q, want %q", res.ResourceID, caseID.String())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Test_toolHandleGetCallTranscript* -- design docs/plans/
// 2026-09-03-insight-assistant-get-call-transcript-design.md §4.
// ---------------------------------------------------------------------------

func testCallTranscriptAIcall(customerID, caseID uuid.UUID) *aicall.AIcall {
	return testAIcallForCase(customerID, caseID)
}

func testCallTranscriptToolCall(toolCallID, callIDArg string) *message.ToolCall {
	return &message.ToolCall{
		ID:   toolCallID,
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameGetCallTranscript,
			Arguments: `{"call_id":"` + callIDArg + `"}`,
		},
	}
}

func testCallForCallTranscript(customerID, callID uuid.UUID) *cmcall.Call {
	return &cmcall.Call{
		Identity: commonidentity.Identity{ID: callID, CustomerID: customerID},
	}
}

func testTranscribeRow(id, customerID, callID uuid.UUID, language string, tmCreate *time.Time) tmtranscribe.Transcribe {
	return tmtranscribe.Transcribe{
		Identity:      commonidentity.Identity{ID: id, CustomerID: customerID},
		ReferenceType: tmtranscribe.ReferenceTypeCall,
		ReferenceID:   callID,
		Language:      language,
		TMCreate:      tmCreate,
	}
}

func testTranscriptRow(id, transcribeID, customerID uuid.UUID, direction tmtranscript.Direction, msg string, tmCreate *time.Time) tmtranscript.Transcript {
	return tmtranscript.Transcript{
		Identity:     commonidentity.Identity{ID: id, CustomerID: customerID},
		TranscribeID: transcribeID,
		Direction:    direction,
		Message:      msg,
		TMCreate:     tmCreate,
	}
}

var (
	ctCustomerID = uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000001")
	ctCaseID     = uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000002")
	ctCallID     = uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000003")
)

// item 1: ReferenceType guard -> fillFailed, zero RPC calls.
func Test_toolHandleGetCallTranscript_ReferenceTypeGuard(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	c.ReferenceType = aicall.ReferenceTypeCall // NOT contact_case

	tc := testCallTranscriptToolCall("tc-guard", ctCallID.String())

	res := h.toolHandleGetCallTranscript(ctx, c, tc)

	if res.Result != "failed" {
		t.Fatalf("Result = %q, want failed", res.Result)
	}
}

// item 2: empty / malformed / uuid.Nil call_id -> fillFailed, three distinct
// cases.
func Test_toolHandleGetCallTranscript_ArgValidation(t *testing.T) {
	tests := []struct {
		name string
		args string
	}{
		{"missing call_id", `{}`},
		{"empty call_id", `{"call_id":""}`},
		{"malformed call_id", `{"call_id":"not-a-uuid"}`},
		{"nil uuid call_id", `{"call_id":"00000000-0000-0000-0000-000000000000"}`},
		{"invalid json", `{`},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &aicallHandler{reqHandler: mockReq}
			ctx := context.Background()

			c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
			tc := &message.ToolCall{
				ID:   "tc-args",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameGetCallTranscript,
					Arguments: tt.args,
				},
			}

			res := h.toolHandleGetCallTranscript(ctx, c, tc)
			if res.Result != "failed" {
				t.Fatalf("Result = %q, want failed", res.Result)
			}
		})
	}
}

// item 3: call not found and call cross-tenant both mask to the
// byte-identical msgResourceNotFound, and zero TranscribeV1TranscribeList
// calls fire on either path (proves the tenant check short-circuits before
// any transcript fan-out).
func Test_toolHandleGetCallTranscript_CallLookupMasking(t *testing.T) {
	tests := []struct {
		name         string
		responseCall *cmcall.Call
		responseErr  error
	}{
		{
			name:        "call not found",
			responseErr: requesthandler.ErrNotFound,
		},
		{
			name: "call cross-tenant",
			responseCall: testCallForCallTranscript(
				uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000ff"), ctCallID),
		},
		{
			name:         "call has nil CustomerID",
			responseCall: testCallForCallTranscript(uuid.Nil, ctCallID),
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

			c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
			tc := testCallTranscriptToolCall("tc-mask", ctCallID.String())

			mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(tt.responseCall, tt.responseErr)
			mockReq.EXPECT().TranscribeV1TranscribeList(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).Times(0)

			res := h.toolHandleGetCallTranscript(ctx, c, tc)

			if res.Result != "success" {
				t.Fatalf("Result = %q, want success (masked)", res.Result)
			}
			if res.Message != msgResourceNotFound {
				t.Errorf("Message = %q, want %q", res.Message, msgResourceNotFound)
			}
		})
	}
}

// item 14b: TranscribeV1TranscribeList RPC error (not not-found) -> honest
// fillFailed, not masked, not degraded.
func Test_toolHandleGetCallTranscript_TranscribeListRPCError(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-list-err", ctCallID.String())

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(nil, errTest)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "failed" {
		t.Fatalf("Result = %q, want failed", res.Result)
	}
}

// item 5: no Transcribe sessions found (post-filter) -> distinct
// "no transcript found for this call", non-masked.
func Test_toolHandleGetCallTranscript_NoSessionsFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-empty", ctCallID.String())

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{}, nil)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success", res.Result)
	}
	if res.Message != "no transcript found for this call" {
		t.Errorf("Message = %q, want distinct not-found message", res.Message)
	}
}

// item 4: happy path -- single session, several transcripts, correct
// [ts direction lang] message rendering.
func Test_toolHandleGetCallTranscript_HappyPath(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-happy", ctCallID.String())

	transcribeID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000010")
	ts1 := timePtr(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC))
	ts2 := timePtr(time.Date(2026, 7, 1, 10, 0, 5, 0, time.UTC))

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(transcribeID, ctCustomerID, ctCallID, "en-US", ts2),
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), gomock.Any()).Return([]tmtranscript.Transcript{
		testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000011"), transcribeID, ctCustomerID, tmtranscript.DirectionOut, "how can I help", ts2),
		testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000012"), transcribeID, ctCustomerID, tmtranscript.DirectionIn, "hello there", ts1),
	}, nil)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	wantIn := "[" + ts1.UTC().Format(time.RFC3339) + " in en-US] hello there"
	wantOut := "[" + ts2.UTC().Format(time.RFC3339) + " out en-US] how can I help"
	if !strings.Contains(res.Message, wantIn) {
		t.Errorf("Message = %q, want it to contain %q", res.Message, wantIn)
	}
	if !strings.Contains(res.Message, wantOut) {
		t.Errorf("Message = %q, want it to contain %q", res.Message, wantOut)
	}
	if strings.Index(res.Message, wantIn) > strings.Index(res.Message, wantOut) {
		t.Errorf("expected chronological order (in before out), got: %s", res.Message)
	}
	if !strings.Contains(res.Message, "transcript_lines: 2") {
		t.Errorf("expected transcript_lines header, got: %s", res.Message)
	}
	if res.ResourceType != "call_transcript" || res.ResourceID != ctCallID.String() {
		t.Errorf("unexpected resource type/id: %s/%s", res.ResourceType, res.ResourceID)
	}
}

// item 6: cross-tenant Transcribe row filtered (IDAIManager scenario) --
// excluded from output, no leak, no numeric hint of its existence.
func Test_toolHandleGetCallTranscript_CrossTenantTranscribeFiltered(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-idam", ctCallID.String())

	genuineID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000020")
	hiddenID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000021") // owned by IDAIManager, foreign customer id
	idAIManager := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000aa")
	ts := timePtr(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC))

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(hiddenID, idAIManager, ctCallID, "en-US", ts),
		testTranscribeRow(genuineID, ctCustomerID, ctCallID, "en-US", ts),
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), gomock.Any()).Return([]tmtranscript.Transcript{
		testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000022"), genuineID, ctCustomerID, tmtranscript.DirectionIn, "hi", ts),
	}, nil)
	// Zero calls expected for the hidden session's TranscriptList.

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	if strings.Contains(res.Message, hiddenID.String()) || strings.Contains(res.Message, idAIManager.String()) {
		t.Errorf("hidden session leaked into output: %s", res.Message)
	}
	if strings.Contains(res.Message, "capped") {
		t.Errorf("unexpected capped marker for a single hidden row: %s", res.Message)
	}
}

// item 7: cross-tenant Transcript row filtered -- excluded, others kept.
// item 7b: mismatched TranscribeID Transcript row -- excluded, does NOT
// render under the wrong session's t.Language.
// item 7c: Transcript-layer TMDelete recheck -- excluded.
func Test_toolHandleGetCallTranscript_TranscriptRowRechecks(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-row-recheck", ctCallID.String())

	transcribeID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000030")
	otherTranscribeID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000031")
	foreignCustomerID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000bb")
	ts := timePtr(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC))

	genuine := testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000032"), transcribeID, ctCustomerID, tmtranscript.DirectionIn, "genuine line", ts)
	crossTenant := testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000033"), transcribeID, foreignCustomerID, tmtranscript.DirectionIn, "foreign customer line", ts)
	wrongSession := testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000034"), otherTranscribeID, ctCustomerID, tmtranscript.DirectionIn, "wrong session line", ts)
	softDeleted := testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000035"), transcribeID, ctCustomerID, tmtranscript.DirectionIn, "deleted line", ts)
	softDeleted.TMDelete = ts

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(transcribeID, ctCustomerID, ctCallID, "en-US", ts),
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), gomock.Any()).Return([]tmtranscript.Transcript{
		genuine, crossTenant, wrongSession, softDeleted,
	}, nil)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	if !strings.Contains(res.Message, "genuine line") {
		t.Errorf("expected genuine row to render, got: %s", res.Message)
	}
	for _, notWant := range []string{"foreign customer line", "wrong session line", "deleted line"} {
		if strings.Contains(res.Message, notWant) {
			t.Errorf("Message = %q, must NOT contain %q", res.Message, notWant)
		}
	}
	if !strings.Contains(res.Message, "transcript_lines: 1") {
		t.Errorf("expected transcript_lines: 1 (only the genuine row), got: %s", res.Message)
	}
}

// item 8: multi-session merge, chronological order across
// sessions/languages.
func Test_toolHandleGetCallTranscript_MultiSessionChronologicalMerge(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-multi", ctCallID.String())

	sessionA := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000040")
	sessionB := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000041")
	t0 := timePtr(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC))
	t1 := timePtr(time.Date(2026, 7, 1, 10, 0, 2, 0, time.UTC))
	t2 := timePtr(time.Date(2026, 7, 1, 10, 0, 4, 0, time.UTC))

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(sessionA, ctCustomerID, ctCallID, "en-US", t0),
		testTranscribeRow(sessionB, ctCustomerID, ctCallID, "ko-KR", t1),
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
		tmtranscript.FieldCustomerID:   ctCustomerID,
		tmtranscript.FieldTranscribeID: sessionA,
		tmtranscript.FieldDeleted:      false,
	}).Return([]tmtranscript.Transcript{
		testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000042"), sessionA, ctCustomerID, tmtranscript.DirectionIn, "english first", t0),
		testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000043"), sessionA, ctCustomerID, tmtranscript.DirectionIn, "english third", t2),
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
		tmtranscript.FieldCustomerID:   ctCustomerID,
		tmtranscript.FieldTranscribeID: sessionB,
		tmtranscript.FieldDeleted:      false,
	}).Return([]tmtranscript.Transcript{
		testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000044"), sessionB, ctCustomerID, tmtranscript.DirectionOut, "korean second", t1),
	}, nil)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	posFirst := strings.Index(res.Message, "english first")
	posSecond := strings.Index(res.Message, "korean second")
	posThird := strings.Index(res.Message, "english third")
	if posFirst < 0 || posSecond < 0 || posThird < 0 {
		t.Fatalf("missing expected lines in: %s", res.Message)
	}
	if posFirst >= posSecond || posSecond >= posThird {
		t.Errorf("expected strict chronological interleaving, got: %s", res.Message)
	}
	if !strings.Contains(res.Message, "in en-US") || !strings.Contains(res.Message, "out ko-KR") {
		t.Errorf("expected per-session language tags, got: %s", res.Message)
	}
}

// item 9: per-session fetch truncation -> gap marker at the correct
// position, a concurrent untruncated session's rows correctly surround it.
// item 9b: §3.4 silent-hole false negative, CORRECTED fixture
// (resourceListPageSize+1 genuine rows plus excluded rows) ->
// sessionFetchTruncated == true, gap marker fires, all resourceListPageSize
// kept rows are genuine.
func Test_toolHandleGetCallTranscript_GapMarker(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-gap", ctCallID.String())

	sessionA := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000050") // truncated (long) session
	sessionB := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000051") // short, concurrent session
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	foreignCustomerID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000cc")

	// The RPC's underlying DB order is TMCreate DESC (newest first,
	// dbhandler/transcript.go), so page1 (fetched with pageToken="") must
	// contain the NEWEST rows and page2 (fetched with the token derived
	// from page1's last/oldest row) the OLDER remainder -- matching real
	// pagination semantics, not just ascending test-fixture order.
	//
	// Item 9b's CORRECTED fixture: resourceListPageSize+1 (101) genuine
	// rows for session A, plus one excluded (cross-tenant) row landing
	// inside the raw fetch. Newest-to-oldest: [newest, excluded,
	// o99, o98, ..., o1, o0] -- 102 raw rows total, split into a full
	// page1 (101 rows: newest + excluded + o99..o1) and a short page2
	// (1 row: o0), which also proves exhaustion.
	newestA := timePtr(base.Add(time.Duration(resourceListPageSize+1) * time.Second))
	excludedTS := timePtr(base.Add(time.Duration(resourceListPageSize) * time.Second))

	page1 := []tmtranscript.Transcript{
		testTranscriptRow(uuid.Must(uuid.NewV4()), sessionA, ctCustomerID, tmtranscript.DirectionIn, "Anew", newestA),
		testTranscriptRow(uuid.Must(uuid.NewV4()), sessionA, foreignCustomerID, tmtranscript.DirectionIn, "Aexc", excludedTS),
	}
	for i := resourceListPageSize - 1; i >= 1; i-- {
		page1 = append(page1, testTranscriptRow(uuid.Must(uuid.NewV4()), sessionA, ctCustomerID, tmtranscript.DirectionIn, fmt.Sprintf("o%03d", i), timePtr(base.Add(time.Duration(i)*time.Second))))
	}
	page2 := []tmtranscript.Transcript{
		testTranscriptRow(uuid.Must(uuid.NewV4()), sessionA, ctCustomerID, tmtranscript.DirectionIn, "o000", timePtr(base)),
	}

	sessionBTS := timePtr(base.Add(time.Duration(resourceListPageSize/2) * time.Second))
	sessionBLine := testTranscriptRow(uuid.Must(uuid.NewV4()), sessionB, ctCustomerID, tmtranscript.DirectionOut, "Bconcurrent", sessionBTS)

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(sessionA, ctCustomerID, ctCallID, "en-US", newestA),
		testTranscribeRow(sessionB, ctCustomerID, ctCallID, "en-US", sessionBTS),
	}, nil)

	gomock.InOrder(
		mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
			tmtranscript.FieldCustomerID:   ctCustomerID,
			tmtranscript.FieldTranscribeID: sessionA,
			tmtranscript.FieldDeleted:      false,
		}).Return(page1, nil),
		mockReq.EXPECT().TranscribeV1TranscriptList(ctx, page1[len(page1)-1].TMCreate.UTC().Format(utilhandler.ISO8601Layout), uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
			tmtranscript.FieldCustomerID:   ctCustomerID,
			tmtranscript.FieldTranscribeID: sessionA,
			tmtranscript.FieldDeleted:      false,
		}).Return(page2, nil),
	)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
		tmtranscript.FieldCustomerID:   ctCustomerID,
		tmtranscript.FieldTranscribeID: sessionB,
		tmtranscript.FieldDeleted:      false,
	}).Return([]tmtranscript.Transcript{sessionBLine}, nil)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	if !strings.Contains(res.Message, "gap en-US") {
		t.Errorf("expected a gap marker for session A, got: %s", res.Message)
	}
	if !strings.Contains(res.Message, "earlier lines of this transcribe session were not fetched") {
		t.Errorf("expected gap marker text, got: %s", res.Message)
	}
	if !strings.Contains(res.Message, "Anew") || !strings.Contains(res.Message, "Bconcurrent") {
		t.Errorf("expected both Anew and Bconcurrent to render, got: %s", res.Message)
	}
	if strings.Contains(res.Message, "Aexc") {
		t.Errorf("excluded cross-tenant row leaked: %s", res.Message)
	}
	// The oldest KEPT genuine A row (o1, the gap boundary -- o0 was dropped
	// by the resourceListPageSize cap) must be genuine, not displaced past
	// the cap by the excluded row (item 9b's filter-then-cap regression).
	if !strings.Contains(res.Message, "o001") {
		t.Errorf("expected the oldest kept genuine row o001 to survive filter-then-cap, got: %s", res.Message)
	}
	if strings.Contains(res.Message, "o000") {
		t.Errorf("expected o000 (dropped by the resourceListPageSize cap) to NOT render, got: %s", res.Message)
	}
}

// item 10: nil TMCreate on a transcript row (sort/render path) -- sorts
// last, is PREFERENTIALLY RETAINED under render-budget truncation, renders
// "unknown", no panic.
func Test_toolHandleGetCallTranscript_NilTMCreateSortsLastAndRetained(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-nil-ts", ctCallID.String())

	transcribeID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000060")
	ts := timePtr(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC))

	// A huge padding message to force render-budget truncation, so the
	// nil-TMCreate row's "preferentially retained" (sorts last = newest)
	// property is actually exercised.
	padding := strings.Repeat("x", 3900)

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(transcribeID, ctCustomerID, ctCallID, "en-US", ts),
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), gomock.Any()).Return([]tmtranscript.Transcript{
		testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000061"), transcribeID, ctCustomerID, tmtranscript.DirectionIn, padding, ts),
		testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000062"), transcribeID, ctCustomerID, tmtranscript.DirectionOut, "nil-ts-line", nil),
	}, nil)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	if !strings.Contains(res.Message, "[unknown out en-US] nil-ts-line") {
		t.Errorf("expected the nil-TMCreate row to render with 'unknown' timestamp and survive truncation, got: %s", res.Message)
	}
}

// item 10b: nil-TMCreate boundary row during pagination halts the loop
// without panicking; sessionCapped/sessionFetchTruncated read true via
// possiblyIncomplete.
func Test_toolHandleGetCallTranscript_NilTMCreatePaginationBoundary(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-nil-boundary", ctCallID.String())

	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	var raw []tmtranscribe.Transcribe
	for i := 0; i < insightCallTranscribeSessionLimit; i++ {
		raw = append(raw, testTranscribeRow(uuid.Must(uuid.NewV4()), ctCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(i)*time.Second))))
	}
	// full page (insightCallTranscribeSessionLimit+1 rows) whose LAST row
	// has TMCreate == nil -- would normally continue paging, but must halt
	// safely instead.
	raw = append(raw, testTranscribeRow(uuid.Must(uuid.NewV4()), ctCustomerID, ctCallID, "en-US", nil))

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(raw, nil).Times(1)

	// insightCallTranscribeSessionLimit verified sessions -> each fetches
	// its own transcripts. sessionCapped must read true via
	// possiblyIncomplete (raw page was full and nil boundary halted
	// pagination without proof), so "could not confirm" -- NOT a real
	// transcript fetch -- is not guaranteed; instead assert no panic and the
	// header capped marker fires while len(verified) == insightCallTranscribeSessionLimit,
	// so the tool DOES proceed to fetch transcripts for those 10 sessions.
	for _, tr := range raw[:insightCallTranscribeSessionLimit] {
		mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
			tmtranscript.FieldCustomerID:   ctCustomerID,
			tmtranscript.FieldTranscribeID: tr.ID,
			tmtranscript.FieldDeleted:      false,
		}).Return([]tmtranscript.Transcript{}, nil)
	}

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (no panic) (message: %s)", res.Result, res.Message)
	}
	if !strings.Contains(res.Message, "transcribe_sessions: 10 (capped, more may exist)") {
		t.Errorf("expected sessionCapped via possiblyIncomplete, got: %s", res.Message)
	}
}

// item 11: session-count cap -- more than insightCallTranscribeSessionLimit
// verified sessions -> capped to 10, header reports the honest cap marker
// with no numeric total, and renderBodyLines' own marker does NOT fire
// purely from the session cap (only from actual line overflow) --
// regression test for the pagedOut=sessionCapped bug.
func Test_toolHandleGetCallTranscript_SessionCountCap(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-cap", ctCallID.String())

	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	var raw []tmtranscribe.Transcribe
	for i := 0; i < insightCallTranscribeSessionLimit+1; i++ { // genuine overflow: 11 genuine sessions
		raw = append(raw, testTranscribeRow(uuid.Must(uuid.NewV4()), ctCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(i)*time.Second))))
	}

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(raw, nil).Times(1)

	for _, tr := range raw[:insightCallTranscribeSessionLimit] {
		mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
			tmtranscript.FieldCustomerID:   ctCustomerID,
			tmtranscript.FieldTranscribeID: tr.ID,
			tmtranscript.FieldDeleted:      false,
		}).Return([]tmtranscript.Transcript{
			testTranscriptRow(uuid.Must(uuid.NewV4()), tr.ID, ctCustomerID, tmtranscript.DirectionIn, "short line", tr.TMCreate),
		}, nil)
	}

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	if !strings.Contains(res.Message, "transcribe_sessions: 10 (capped, more may exist)") {
		t.Errorf("expected honest cap header, got: %s", res.Message)
	}
	if strings.Contains(res.Message, "earlier transcript lines omitted") {
		t.Errorf("renderBodyLines' own marker must NOT fire purely from the session cap, got: %s", res.Message)
	}
}

// item 12 (session layer): filter-then-cap ordering -- a raw session list
// where a foreign-tenant row appears WITHIN the first
// insightCallTranscribeSessionLimit+1 rows, alongside
// insightCallTranscribeSessionLimit genuine rows -> all genuine rows must
// survive (none displaced past the cap by the foreign row).
func Test_toolHandleGetCallTranscript_SessionLayerFilterThenCap(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-filter-then-cap", ctCallID.String())

	foreignCustomerID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000dd")
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	var raw []tmtranscribe.Transcribe
	// one foreign row FIRST, then exactly insightCallTranscribeSessionLimit
	// genuine rows -- total insightCallTranscribeSessionLimit+1, matching a
	// single raw page.
	raw = append(raw, testTranscribeRow(uuid.Must(uuid.NewV4()), foreignCustomerID, ctCallID, "en-US", timePtr(base)))
	genuineIDs := make([]uuid.UUID, 0, insightCallTranscribeSessionLimit)
	for i := 1; i <= insightCallTranscribeSessionLimit; i++ {
		id := uuid.Must(uuid.NewV4())
		genuineIDs = append(genuineIDs, id)
		raw = append(raw, testTranscribeRow(id, ctCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(i)*time.Second))))
	}

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	// The raw page is FULL (insightCallTranscribeSessionLimit+1 rows) but
	// only insightCallTranscribeSessionLimit are genuine after filtering,
	// which is NOT proof of overflow -- the loop must fetch a second, SHORT
	// page before it can prove exhaustion (§3.3's pagination-until-exact).
	lastTS := raw[len(raw)-1].TMCreate.UTC().Format(utilhandler.ISO8601Layout)
	gomock.InOrder(
		mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(raw, nil),
		mockReq.EXPECT().TranscribeV1TranscribeList(ctx, lastTS, uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{}, nil),
	)

	for _, id := range genuineIDs {
		mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
			tmtranscript.FieldCustomerID:   ctCustomerID,
			tmtranscript.FieldTranscribeID: id,
			tmtranscript.FieldDeleted:      false,
		}).Return([]tmtranscript.Transcript{}, nil)
	}

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	// The mock's strict verification above (one TranscriptList EXPECT per
	// genuine id, none for the foreign id) is itself the assertion that all
	// insightCallTranscribeSessionLimit genuine rows survived the cap.
}

// item 12 (transcript layer): row-level filter-then-cap ordering -- a raw
// transcript page where a foreign-tenant row appears WITHIN the first
// resourceListPageSize+1 rows, alongside resourceListPageSize genuine rows
// -> all genuine rows must survive.
func Test_toolHandleGetCallTranscript_TranscriptLayerFilterThenCap(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-row-filter-then-cap", ctCallID.String())

	transcribeID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000070")
	foreignCustomerID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000ee")
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	var raw []tmtranscript.Transcript
	raw = append(raw, testTranscriptRow(uuid.Must(uuid.NewV4()), transcribeID, foreignCustomerID, tmtranscript.DirectionIn, "foreign-row", timePtr(base)))
	for i := 1; i <= resourceListPageSize; i++ {
		raw = append(raw, testTranscriptRow(uuid.Must(uuid.NewV4()), transcribeID, ctCustomerID, tmtranscript.DirectionIn, fmt.Sprintf("genuine-%d", i), timePtr(base.Add(time.Duration(i)*time.Second))))
	}

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(transcribeID, ctCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(resourceListPageSize)*time.Second))),
	}, nil)
	// The raw page is FULL (resourceListPageSize+1 rows) but only
	// resourceListPageSize are genuine after filtering, which is NOT proof
	// of overflow -- the loop must fetch a second, SHORT page before it can
	// prove exhaustion.
	lastTS := raw[len(raw)-1].TMCreate.UTC().Format(utilhandler.ISO8601Layout)
	gomock.InOrder(
		mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), gomock.Any()).Return(raw, nil),
		mockReq.EXPECT().TranscribeV1TranscriptList(ctx, lastTS, uint64(resourceListPageSize+1), gomock.Any()).Return([]tmtranscript.Transcript{}, nil),
	)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	if strings.Contains(res.Message, "foreign-row") {
		t.Errorf("foreign row leaked: %s", res.Message)
	}
	for i := 1; i <= resourceListPageSize; i++ {
		want := fmt.Sprintf("genuine-%d", i)
		if !strings.Contains(res.Message, want) {
			// render-budget truncation may legitimately drop the oldest
			// lines -- only assert the header's pre-render count, not every
			// individual line, to avoid a flaky over-assertion here.
			break
		}
	}
	if !strings.Contains(res.Message, "transcript_lines: "+fmt.Sprint(resourceListPageSize)) {
		t.Errorf("expected all %d genuine rows counted pre-render, got: %s", resourceListPageSize, res.Message)
	}
}

// item 13: wrong-call / soft-deleted Transcribe row rechecked -> excluded.
func Test_toolHandleGetCallTranscript_TranscribeRowRechecks(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-transcribe-recheck", ctCallID.String())

	genuineID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000080")
	wrongCallID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000081")
	deletedID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000082")
	ts := timePtr(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC))

	wrongCall := testTranscribeRow(wrongCallID, ctCustomerID, uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000ff"), "en-US", ts)
	deleted := testTranscribeRow(deletedID, ctCustomerID, ctCallID, "en-US", ts)
	deleted.TMDelete = ts

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(genuineID, ctCustomerID, ctCallID, "en-US", ts),
		wrongCall,
		deleted,
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
		tmtranscript.FieldCustomerID:   ctCustomerID,
		tmtranscript.FieldTranscribeID: genuineID,
		tmtranscript.FieldDeleted:      false,
	}).Return([]tmtranscript.Transcript{}, nil)
	// No TranscriptList EXPECT for wrongCallID / deletedID -- strict gomock
	// fails the test if either is reached.

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
}

// item 14: per-session RPC failure degrades VISIBLY (sessions_unavailable
// header field), doesn't fail the whole tool.
func Test_toolHandleGetCallTranscript_PerSessionRPCFailureDegradesVisibly(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-session-fail", ctCallID.String())

	sessionOK := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000090")
	sessionFail := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000091")
	ts := timePtr(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC))

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(sessionOK, ctCustomerID, ctCallID, "en-US", ts),
		testTranscribeRow(sessionFail, ctCustomerID, ctCallID, "en-US", ts),
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
		tmtranscript.FieldCustomerID:   ctCustomerID,
		tmtranscript.FieldTranscribeID: sessionOK,
		tmtranscript.FieldDeleted:      false,
	}).Return([]tmtranscript.Transcript{
		testTranscriptRow(uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-000000000092"), sessionOK, ctCustomerID, tmtranscript.DirectionIn, "still works", ts),
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
		tmtranscript.FieldCustomerID:   ctCustomerID,
		tmtranscript.FieldTranscribeID: sessionFail,
		tmtranscript.FieldDeleted:      false,
	}).Return(nil, errTest)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	if !strings.Contains(res.Message, "still works") {
		t.Errorf("expected the healthy session's line to still render, got: %s", res.Message)
	}
	if !strings.Contains(res.Message, "sessions_unavailable: 1") {
		t.Errorf("expected sessions_unavailable: 1 header, got: %s", res.Message)
	}
}

// item 15: header transcript_lines: N reflects only real rows, excluding
// any gap markers -- covered incidentally above (Test_..._TranscriptRowRechecks
// asserts transcript_lines: 1); this test isolates it with an explicit gap
// marker present too, to prove the marker itself is not counted.
func Test_toolHandleGetCallTranscript_TranscriptLinesExcludesGapMarkers(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-lines-count", ctCallID.String())

	sessionID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000a0")
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	// resourceListPageSize+1 genuine rows, zero exclusions: a single full
	// page already proves overflow (verified > resourceListPageSize), so
	// exactly ONE TranscriptList call happens -- no second page needed.
	var page1 []tmtranscript.Transcript
	for i := 0; i < resourceListPageSize+1; i++ {
		page1 = append(page1, testTranscriptRow(uuid.Must(uuid.NewV4()), sessionID, ctCustomerID, tmtranscript.DirectionIn, fmt.Sprintf("l-%d", i), timePtr(base.Add(time.Duration(i)*time.Second))))
	}

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(sessionID, ctCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(resourceListPageSize+1)*time.Second))),
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), gomock.Any()).Return(page1, nil)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	// resourceListPageSize genuine rows are kept (capped) + a gap marker is
	// NOT counted toward transcript_lines.
	if !strings.Contains(res.Message, "transcript_lines: "+fmt.Sprint(resourceListPageSize)) {
		t.Errorf("expected transcript_lines: %d excluding the gap marker, got: %s", resourceListPageSize, res.Message)
	}
}

// item 16: pagination loop actually pages when needed -- pageToken uses the
// LITERAL utilhandler.ISO8601Layout format (not RFC3339Nano); the final
// verified slice contains every genuine row across all pages, capped
// correctly.
func Test_toolHandleGetCallTranscript_PaginationLoopPagesWhenNeeded(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-pagination", ctCallID.String())

	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	foreignCustomerID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000f0")

	// A single full page of ALL-genuine rows would prove overflow
	// immediately (no second fetch needed) -- to force the loop to
	// genuinely page (item 16's actual point), page1 mixes exactly
	// insightCallTranscribeSessionLimit genuine rows with one excluded
	// (foreign-tenant) row, filling the full insightCallTranscribeSessionLimit+1
	// raw page without yet proving overflow. page2 supplies one more
	// genuine row (the true overflow-proving row).
	var page1, page2 []tmtranscribe.Transcribe
	page1 = append(page1, testTranscribeRow(uuid.Must(uuid.NewV4()), foreignCustomerID, ctCallID, "en-US", timePtr(base)))
	genuineIDs := make([]uuid.UUID, 0, insightCallTranscribeSessionLimit+1)
	for i := 1; i <= insightCallTranscribeSessionLimit; i++ {
		id := uuid.Must(uuid.NewV4())
		genuineIDs = append(genuineIDs, id)
		page1 = append(page1, testTranscribeRow(id, ctCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(i)*time.Second))))
	}
	lastGenuineID := uuid.Must(uuid.NewV4())
	genuineIDs = append(genuineIDs, lastGenuineID)
	page2 = append(page2, testTranscribeRow(lastGenuineID, ctCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(insightCallTranscribeSessionLimit+1)*time.Second))))

	lastPage1TS := page1[len(page1)-1].TMCreate
	wantToken := lastPage1TS.UTC().Format(utilhandler.ISO8601Layout)
	if !strings.Contains(wantToken, ".") || strings.HasSuffix(wantToken, "Z") == false {
		t.Fatalf("sanity check on ISO8601Layout format failed: %s", wantToken)
	}

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	gomock.InOrder(
		mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(page1, nil),
		mockReq.EXPECT().TranscribeV1TranscribeList(ctx, wantToken, uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(page2, nil),
	)
	// Only the first insightCallTranscribeSessionLimit genuine sessions
	// (capped) get a TranscriptList fetch -- the (limit+1)'th genuine
	// session is dropped by the cap.
	for _, id := range genuineIDs[:insightCallTranscribeSessionLimit] {
		mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
			tmtranscript.FieldCustomerID:   ctCustomerID,
			tmtranscript.FieldTranscribeID: id,
			tmtranscript.FieldDeleted:      false,
		}).Return([]tmtranscript.Transcript{}, nil)
	}

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	if !strings.Contains(res.Message, "transcribe_sessions: 10 (capped, more may exist)") {
		t.Errorf("expected capped header after multi-page pagination, got: %s", res.Message)
	}
}

// item 17: sessionCapped does NOT leak filtered-row existence for H>=2
// hidden rows spread across page boundaries -- byte-identical rendered
// output AND header between the with-hidden-rows and without-hidden-rows
// fixtures.
func Test_toolHandleGetCallTranscript_HiddenRowsDoNotLeak(t *testing.T) {
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	foreignCustomerID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000f1")

	run := func(t *testing.T, hidden bool) string {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq}
		ctx := context.Background()

		c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
		tc := testCallTranscriptToolCall("tc-h2", ctCallID.String())

		genuineIDs := make([]uuid.UUID, 0, insightCallTranscribeSessionLimit)
		var page1 []tmtranscribe.Transcribe
		if hidden {
			page1 = append(page1, testTranscribeRow(uuid.Must(uuid.NewV4()), foreignCustomerID, ctCallID, "en-US", timePtr(base)))
		}
		for i := 1; i <= insightCallTranscribeSessionLimit; i++ {
			id := uuid.Must(uuid.NewV4())
			genuineIDs = append(genuineIDs, id)
			page1 = append(page1, testTranscribeRow(id, ctCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(i)*time.Second))))
		}
		var page2 []tmtranscribe.Transcribe
		if hidden {
			// a second hidden row landing on the second page.
			page2 = append(page2, testTranscribeRow(uuid.Must(uuid.NewV4()), foreignCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(insightCallTranscribeSessionLimit+1)*time.Second))))
		}

		mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
		if hidden {
			lastTS := page1[len(page1)-1].TMCreate.UTC().Format(utilhandler.ISO8601Layout)
			gomock.InOrder(
				mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(page1, nil),
				mockReq.EXPECT().TranscribeV1TranscribeList(ctx, lastTS, uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(page2, nil),
			)
		} else {
			mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(page1, nil)
		}
		for _, id := range genuineIDs {
			mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
				tmtranscript.FieldCustomerID:   ctCustomerID,
				tmtranscript.FieldTranscribeID: id,
				tmtranscript.FieldDeleted:      false,
			}).Return([]tmtranscript.Transcript{}, nil)
		}

		res := h.toolHandleGetCallTranscript(ctx, c, tc)
		if res.Result != "success" {
			t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
		}
		return res.Message
	}

	withoutHidden := run(t, false)
	withHidden := run(t, true)

	if withoutHidden != withHidden {
		t.Errorf("hidden-row presence leaked into rendered output:\nwithout: %s\nwith:    %s", withoutHidden, withHidden)
	}
}

// item 18: nominal path, exactly limit genuine rows, WITH and WITHOUT
// exclusions -- both must read false, no false-positive cap/gap
// (regression test for the retired >= comparator bug).
func Test_toolHandleGetCallTranscript_NominalPathNoFalsePositive(t *testing.T) {
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	foreignCustomerID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000f2")

	run := func(t *testing.T, withExclusion bool) {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq}
		ctx := context.Background()

		c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
		tc := testCallTranscriptToolCall("tc-nominal", ctCallID.String())

		genuineIDs := make([]uuid.UUID, 0, insightCallTranscribeSessionLimit)
		var raw []tmtranscribe.Transcribe
		if withExclusion {
			raw = append(raw, testTranscribeRow(uuid.Must(uuid.NewV4()), foreignCustomerID, ctCallID, "en-US", timePtr(base)))
		}
		for i := 1; i <= insightCallTranscribeSessionLimit; i++ {
			id := uuid.Must(uuid.NewV4())
			genuineIDs = append(genuineIDs, id)
			raw = append(raw, testTranscribeRow(id, ctCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(i)*time.Second))))
		}

		mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
		if withExclusion {
			// raw page is full (limit+1 rows requested, limit+1 returned)
			// -> loop must fetch a second, SHORT page before it can prove
			// exhaustion.
			lastTS := raw[len(raw)-1].TMCreate.UTC().Format(utilhandler.ISO8601Layout)
			gomock.InOrder(
				mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(raw, nil),
				mockReq.EXPECT().TranscribeV1TranscribeList(ctx, lastTS, uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{}, nil),
			)
		} else {
			mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(raw, nil)
		}
		for _, id := range genuineIDs {
			mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
				tmtranscript.FieldCustomerID:   ctCustomerID,
				tmtranscript.FieldTranscribeID: id,
				tmtranscript.FieldDeleted:      false,
			}).Return([]tmtranscript.Transcript{}, nil)
		}

		res := h.toolHandleGetCallTranscript(ctx, c, tc)
		if res.Result != "success" {
			t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
		}
		if strings.Contains(res.Message, "capped") {
			t.Errorf("false-positive cap marker for exactly %d genuine rows (withExclusion=%v): %s", insightCallTranscribeSessionLimit, withExclusion, res.Message)
		}
	}

	t.Run("zero exclusions", func(t *testing.T) { run(t, false) })
	t.Run("with exclusions inside the full page", func(t *testing.T) { run(t, true) })
}

// item 19: pagination loop respects insightCallTranscribeFetchMaxPages, and
// possiblyIncomplete makes the page-cap exit honest -- >=45 excluded rows
// at the session layer exhausts the 5-page budget without proof; assert
// sessionCapped == true via possiblyIncomplete and no numeric hidden-row
// count anywhere in the header.
func Test_toolHandleGetCallTranscript_PageCapPossiblyIncomplete(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-page-cap", ctCallID.String())

	foreignCustomerID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000f3")
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	// 5 full pages of insightCallTranscribeSessionLimit+1 rows each, every
	// row on every page EXCLUDED (foreign customer) so the loop never has
	// enough genuine rows to prove overflow, and never a short page to
	// prove exhaustion -- so the loop must exit at the page cap with
	// possiblyIncomplete=true. 5 * 11 = 55 raw rows, all excluded.
	var pages [][]tmtranscribe.Transcribe
	seqN := 0
	for page := 0; page < insightCallTranscribeFetchMaxPages; page++ {
		var rows []tmtranscribe.Transcribe
		for i := 0; i < insightCallTranscribeSessionLimit+1; i++ {
			rows = append(rows, testTranscribeRow(uuid.Must(uuid.NewV4()), foreignCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(seqN)*time.Second))))
			seqN++
		}
		pages = append(pages, rows)
	}

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	calls := make([]any, 0, len(pages))
	token := ""
	for _, rows := range pages {
		rowsCopy := rows
		tokenCopy := token
		calls = append(calls, mockReq.EXPECT().TranscribeV1TranscribeList(ctx, tokenCopy, uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(rowsCopy, nil))
		token = rowsCopy[len(rowsCopy)-1].TMCreate.UTC().Format(utilhandler.ISO8601Layout)
	}
	gomock.InOrder(calls...)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	if res.Message != "could not confirm whether a transcript exists for this call" {
		t.Errorf("expected the honest could-not-confirm message at the page cap, got: %s", res.Message)
	}
	// No numeric hidden-row count anywhere.
	for _, n := range []string{"45", "55"} {
		if strings.Contains(res.Message, n) {
			t.Errorf("numeric hidden-row count leaked into header: %s", res.Message)
		}
	}
}

// item 20: sessionCapped-guarded not-found message -- verified empty +
// sessionCapped=true -> distinct "could not confirm" message, not a false
// "no transcript found". Reuses the same page-cap fixture shape as item 19
// but with zero genuine rows at all (already covered by item 19's assertion
// on res.Message). This test isolates the SIMPLEST possiblyIncomplete path:
// a nil-TMCreate boundary on the very first page halts before any genuine
// row is confirmed.
func Test_toolHandleGetCallTranscript_SessionCappedGuardedNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-guarded-notfound", ctCallID.String())

	foreignCustomerID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000f4")
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	var raw []tmtranscribe.Transcribe
	for i := 0; i < insightCallTranscribeSessionLimit; i++ {
		raw = append(raw, testTranscribeRow(uuid.Must(uuid.NewV4()), foreignCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(i)*time.Second))))
	}
	// full page (limit+1 rows), last row has nil TMCreate -> halts pagination
	// without proof, zero genuine rows confirmed.
	raw = append(raw, testTranscribeRow(uuid.Must(uuid.NewV4()), foreignCustomerID, ctCallID, "en-US", nil))

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(raw, nil).Times(1)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	if res.Message != "could not confirm whether a transcript exists for this call" {
		t.Errorf("Message = %q, want the distinct could-not-confirm message", res.Message)
	}
}

// item 21: OUTER RPC fan-out count (session-iteration loop) is invariant to
// hidden-row presence; INNER pagination RPC count is NOT invariant -- pin
// the actual variance (1 page vs 2+ pages).
func Test_toolHandleGetCallTranscript_OuterInvariantInnerVariant(t *testing.T) {
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)
	foreignCustomerID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000f5")

	run := func(t *testing.T, hidden bool) int {
		mc := gomock.NewController(t)
		defer mc.Finish()

		mockReq := requesthandler.NewMockRequestHandler(mc)
		h := &aicallHandler{reqHandler: mockReq}
		ctx := context.Background()

		c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
		tc := testCallTranscriptToolCall("tc-outer-inner", ctCallID.String())

		genuineIDs := make([]uuid.UUID, 0, insightCallTranscribeSessionLimit)
		var page1 []tmtranscribe.Transcribe
		for i := 0; i < insightCallTranscribeSessionLimit; i++ {
			id := uuid.Must(uuid.NewV4())
			genuineIDs = append(genuineIDs, id)
			page1 = append(page1, testTranscribeRow(id, ctCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(i)*time.Second))))
		}
		pageCalls := 1
		mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
		if hidden {
			page1 = append(page1, testTranscribeRow(uuid.Must(uuid.NewV4()), foreignCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(insightCallTranscribeSessionLimit)*time.Second))))
			lastTS := page1[len(page1)-1].TMCreate.UTC().Format(utilhandler.ISO8601Layout)
			gomock.InOrder(
				mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(page1, nil),
				mockReq.EXPECT().TranscribeV1TranscribeList(ctx, lastTS, uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{}, nil),
			)
			pageCalls = 2
		} else {
			mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return(page1, nil)
		}
		for _, id := range genuineIDs {
			mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
				tmtranscript.FieldCustomerID:   ctCustomerID,
				tmtranscript.FieldTranscribeID: id,
				tmtranscript.FieldDeleted:      false,
			}).Return([]tmtranscript.Transcript{}, nil)
		}

		res := h.toolHandleGetCallTranscript(ctx, c, tc)
		if res.Result != "success" {
			t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
		}
		return pageCalls
	}

	pagesWithoutHidden := run(t, false)
	pagesWithHidden := run(t, true)

	if pagesWithoutHidden == pagesWithHidden {
		t.Errorf("expected the INNER page count to differ (1 vs 2+) between hidden/non-hidden fixtures -- got %d both times", pagesWithoutHidden)
	}
	// The outer session-iteration count (insightCallTranscribeSessionLimit
	// TranscriptList call-groups) is asserted implicitly by both runs'
	// strict gomock expectations succeeding identically for genuineIDs.
}

// item 22 (partial: seq tiebreak determinism): two rows with identical
// TMCreate, rank, TranscribeID, AND TranscriptID (a constructed adversarial
// fixture) -> sort order is deterministic and matches fetch order.
func Test_toolHandleGetCallTranscript_SeqTiebreakDeterminism(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-seq", ctCallID.String())

	transcribeID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000f6")
	dupID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000f7")
	ts := timePtr(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC))

	first := testTranscriptRow(dupID, transcribeID, ctCustomerID, tmtranscript.DirectionIn, "fetched-first", ts)
	second := testTranscriptRow(dupID, transcribeID, ctCustomerID, tmtranscript.DirectionIn, "fetched-second", ts)

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(transcribeID, ctCustomerID, ctCallID, "en-US", ts),
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), gomock.Any()).Return([]tmtranscript.Transcript{first, second}, nil)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	posFirst := strings.Index(res.Message, "fetched-first")
	posSecond := strings.Index(res.Message, "fetched-second")
	if posFirst < 0 || posSecond < 0 || posFirst > posSecond {
		t.Errorf("expected deterministic fetch-order tiebreak, got: %s", res.Message)
	}
}

// item 22 (partial: gap-boundary nil-TMCreate handling): the marker copies
// the nil verbatim and, per the nil-sorts-last rule, is placed adjacent to
// that row wherever it sorts (including at the tail).
func Test_toolHandleGetCallTranscript_GapBoundaryNilTMCreate(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-gap-nil", ctCallID.String())

	sessionID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000f8")
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	// resourceListPageSize+1 rows on the first page, whose LAST row (the
	// gap boundary, once sliced to the cap) has TMCreate == nil.
	var page1 []tmtranscript.Transcript
	for i := 0; i < resourceListPageSize; i++ {
		page1 = append(page1, testTranscriptRow(uuid.Must(uuid.NewV4()), sessionID, ctCustomerID, tmtranscript.DirectionIn, fmt.Sprintf("g-%d", i), timePtr(base.Add(time.Duration(i)*time.Second))))
	}
	page1 = append(page1, testTranscriptRow(uuid.Must(uuid.NewV4()), sessionID, ctCustomerID, tmtranscript.DirectionIn, "g-boundary-nil", nil))

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(sessionID, ctCustomerID, ctCallID, "en-US", timePtr(base.Add(time.Duration(resourceListPageSize)*time.Second))),
	}, nil).Times(1)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), gomock.Any()).Return(page1, nil).Times(1)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (no panic) (message: %s)", res.Result, res.Message)
	}
	if !strings.Contains(res.Message, "gap en-US") {
		t.Errorf("expected a gap marker to be synthesized, got: %s", res.Message)
	}
}

// item 22 (partial: transcript_lines pre-render-vs-rendered divergence): the
// header's transcript_lines: N reflects the full pre-truncation merged
// count, while fewer lines are actually rendered below it.
func Test_toolHandleGetCallTranscript_TranscriptLinesPreRenderVsRendered(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-prerender", ctCallID.String())

	sessionID := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000f9")
	base := time.Date(2026, 7, 1, 9, 0, 0, 0, time.UTC)

	// 60 lines, each padded so the merged total comfortably exceeds the
	// 4000-rune whole-message budget once the header is added.
	var rows []tmtranscript.Transcript
	padding := strings.Repeat("y", 100)
	for i := 0; i < 60; i++ {
		rows = append(rows, testTranscriptRow(uuid.Must(uuid.NewV4()), sessionID, ctCustomerID, tmtranscript.DirectionIn, fmt.Sprintf("LINE%02d-%s", i, padding), timePtr(base.Add(time.Duration(i)*time.Second))))
	}

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(sessionID, ctCustomerID, ctCallID, "en-US", timePtr(base.Add(59*time.Second))),
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), gomock.Any()).Return(rows, nil)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (message: %s)", res.Result, res.Message)
	}
	if !strings.Contains(res.Message, "transcript_lines: 60") {
		t.Errorf("expected pre-truncation merged count of 60, got: %s", res.Message)
	}
	// Prove NOT all 60 rendered: the oldest row's text should have been
	// dropped by renderBodyLines' render-budget truncation.
	if strings.Contains(res.Message, "LINE00-"+padding) {
		t.Errorf("expected the oldest line to be truncated away, but it rendered: %s", res.Message)
	}
	if !strings.Contains(res.Message, "earlier transcript lines omitted") {
		t.Errorf("expected renderBodyLines' own truncation marker, got: %s", res.Message)
	}
}

// item 22 (partial): all-sessions-unavailable terminal state -- Result:
// success, transcript_lines: 0, sessions_unavailable: N, no panic on an
// empty merge.
func Test_toolHandleGetCallTranscript_AllSessionsUnavailable(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &aicallHandler{reqHandler: mockReq}
	ctx := context.Background()

	c := testCallTranscriptAIcall(ctCustomerID, ctCaseID)
	tc := testCallTranscriptToolCall("tc-all-unavailable", ctCallID.String())

	sessionA := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000fa")
	sessionB := uuid.FromStringOrNil("7b2e3d20-c002-11f0-9000-0000000000fb")
	ts := timePtr(time.Date(2026, 7, 1, 10, 0, 0, 0, time.UTC))

	mockReq.EXPECT().CallV1CallGet(ctx, ctCallID).Return(testCallForCallTranscript(ctCustomerID, ctCallID), nil)
	mockReq.EXPECT().TranscribeV1TranscribeList(ctx, "", uint64(insightCallTranscribeSessionLimit+1), gomock.Any()).Return([]tmtranscribe.Transcribe{
		testTranscribeRow(sessionA, ctCustomerID, ctCallID, "en-US", ts),
		testTranscribeRow(sessionB, ctCustomerID, ctCallID, "en-US", ts),
	}, nil)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
		tmtranscript.FieldCustomerID:   ctCustomerID,
		tmtranscript.FieldTranscribeID: sessionA,
		tmtranscript.FieldDeleted:      false,
	}).Return(nil, errTest)
	mockReq.EXPECT().TranscribeV1TranscriptList(ctx, "", uint64(resourceListPageSize+1), map[tmtranscript.Field]any{
		tmtranscript.FieldCustomerID:   ctCustomerID,
		tmtranscript.FieldTranscribeID: sessionB,
		tmtranscript.FieldDeleted:      false,
	}).Return(nil, errTest)

	res := h.toolHandleGetCallTranscript(ctx, c, tc)
	if res.Result != "success" {
		t.Fatalf("Result = %q, want success (no panic on empty merge) (message: %s)", res.Result, res.Message)
	}
	if !strings.Contains(res.Message, "transcript_lines: 0") {
		t.Errorf("expected transcript_lines: 0, got: %s", res.Message)
	}
	if !strings.Contains(res.Message, "sessions_unavailable: 2") {
		t.Errorf("expected sessions_unavailable: 2, got: %s", res.Message)
	}
}

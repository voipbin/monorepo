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
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	commonoutline "monorepo/bin-common-handler/models/outline"
	"monorepo/bin-common-handler/pkg/requesthandler"

	cmcasenote "monorepo/bin-contact-manager/models/casenote"
	cmcontact "monorepo/bin-contact-manager/models/contact"
	kmkase "monorepo/bin-contact-manager/models/kase"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
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

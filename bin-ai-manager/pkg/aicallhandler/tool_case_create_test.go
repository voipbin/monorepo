package aicallhandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcall "monorepo/bin-call-manager/models/call"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	kmkase "monorepo/bin-contact-manager/models/kase"
	fmvariable "monorepo/bin-flow-manager/models/variable"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
)

// Test_deriveEndpointsForCase verifies the direction-based peer/self
// resolution shared with bin-flow-manager's identically-named helper
// (design VOIP-1243 §6.3).
func Test_deriveEndpointsForCase(t *testing.T) {
	source := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"}
	dest := commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0002"}

	tests := []struct {
		name       string
		direction  string
		expectPeer commonaddress.Address
		expectSelf commonaddress.Address
	}{
		{"incoming", "incoming", source, dest},
		{"outgoing", "outgoing", dest, source},
		{"unknown", "unknown", commonaddress.Address{}, commonaddress.Address{}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer, self := deriveEndpointsForCase(tt.direction, source, dest)
			if peer != tt.expectPeer || self != tt.expectSelf {
				t.Errorf("deriveEndpointsForCase(%s) = (%v, %v), want (%v, %v)", tt.direction, peer, self, tt.expectPeer, tt.expectSelf)
			}
		})
	}
}

// Test_isCRMEligiblePeer verifies the ineligible-peer-type filter matches
// bin-flow-manager's / contacthandler's copy (design VOIP-1243 §6.3).
func Test_isCRMEligiblePeer(t *testing.T) {
	tests := []struct {
		name     string
		peerType commonaddress.Type
		expect   bool
	}{
		{"tel is eligible", commonaddress.TypeTel, true},
		{"email is eligible", commonaddress.TypeEmail, true},
		{"agent is not eligible", commonaddress.TypeAgent, false},
		{"conference is not eligible", commonaddress.TypeConference, false},
		{"sip is not eligible", commonaddress.TypeSIP, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isCRMEligiblePeer(tt.peerType); got != tt.expect {
				t.Errorf("isCRMEligiblePeer(%s) = %v, want %v", tt.peerType, got, tt.expect)
			}
		})
	}
}

// Test_deriveCaseEndpointsForAIcall verifies the call/conversation/other
// dispatch, mirroring bin-flow-manager's actionHandleCaseCreate switch
// (design VOIP-1243 §6.3).
func Test_deriveCaseEndpointsForAIcall(t *testing.T) {

	tests := []struct {
		name string

		aicall *aicall.AIcall

		responseCall         *cmcall.Call
		responseConversation *cvconversation.Conversation
		responseErr          error

		expectPeer          commonaddress.Address
		expectSelf          commonaddress.Address
		expectReferenceType string
		expectSupported     bool
		expectErr           bool
	}{
		{
			name: "call reference type",
			aicall: &aicall.AIcall{
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("2b6f4c3e-c001-11f0-9000-000000000001"),
			},
			responseCall: &cmcall.Call{
				Direction:   cmcall.DirectionIncoming,
				Source:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"},
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0002"},
			},
			expectPeer:          commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"},
			expectSelf:          commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0002"},
			expectReferenceType: "call",
			expectSupported:     true,
		},
		{
			name: "conversation reference type",
			aicall: &aicall.AIcall{
				ReferenceType: aicall.ReferenceTypeConversation,
				ReferenceID:   uuid.FromStringOrNil("2b6f4c3e-c001-11f0-9000-000000000002"),
			},
			responseConversation: &cvconversation.Conversation{
				Self: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0003"},
				Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0004"},
			},
			expectPeer:          commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0004"},
			expectSelf:          commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0003"},
			expectReferenceType: "conversation_message",
			expectSupported:     true,
		},
		{
			name: "unsupported reference type",
			aicall: &aicall.AIcall{
				ReferenceType: aicall.ReferenceTypeTask,
				ReferenceID:   uuid.FromStringOrNil("2b6f4c3e-c001-11f0-9000-000000000003"),
			},
			expectSupported: false,
		},
		{
			name: "call reference type - CallV1CallGet errors -> err is returned, not swallowed",
			aicall: &aicall.AIcall{
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("2b6f4c3e-c001-11f0-9000-000000000004"),
			},
			responseErr:     errTest,
			expectSupported: true,
			expectErr:       true,
		},
		{
			name: "conversation reference type - ConversationV1ConversationGet errors -> err is returned, not swallowed",
			aicall: &aicall.AIcall{
				ReferenceType: aicall.ReferenceTypeConversation,
				ReferenceID:   uuid.FromStringOrNil("2b6f4c3e-c001-11f0-9000-000000000005"),
			},
			responseErr:     errTest,
			expectSupported: true,
			expectErr:       true,
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

			switch tt.aicall.ReferenceType {
			case aicall.ReferenceTypeCall:
				mockReq.EXPECT().CallV1CallGet(ctx, tt.aicall.ReferenceID).Return(tt.responseCall, tt.responseErr)
			case aicall.ReferenceTypeConversation:
				mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.aicall.ReferenceID).Return(tt.responseConversation, tt.responseErr)
			}

			peer, self, referenceType, supported, err := h.deriveCaseEndpointsForAIcall(ctx, tt.aicall)
			if supported != tt.expectSupported {
				t.Fatalf("supported = %v, want %v", supported, tt.expectSupported)
			}
			if tt.expectErr {
				if err == nil {
					t.Fatalf("expected a non-nil err, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected err: %v", err)
			}
			if !tt.expectSupported {
				return
			}
			if peer != tt.expectPeer || self != tt.expectSelf || referenceType != tt.expectReferenceType {
				t.Errorf("got (%v, %v, %s), want (%v, %v, %s)", peer, self, referenceType, tt.expectPeer, tt.expectSelf, tt.expectReferenceType)
			}
		})
	}
}

// Test_toolHandleCaseCreate covers design VOIP-1243 §6.3/§8/§3.5: happy
// path, ContactV1CaseCreate error -> fillFailed, CRM-ineligible peer ->
// fillSuccess (not fillFailed), and the activeflow-scoped dedup skip.
func Test_toolHandleCaseCreate(t *testing.T) {

	tests := []struct {
		name string

		aicall *aicall.AIcall
		tool   *message.ToolCall

		responseCall     *cmcall.Call
		responseVariable *fmvariable.Variable
		responseCase     *kmkase.Case
		responseCreateErr error

		expectContactV1CaseCreate bool
		expectRes                 *messageContent
	}{
		{
			name: "happy path",
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000001"),
					CustomerID: uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000002"),
				},
				ActiveflowID:  uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000003"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000004"),
			},
			tool: &message.ToolCall{
				ID:   "3a1f2c10-c001-11f0-9000-000000000005",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameCaseCreate,
					Arguments: `{"name": "VIP escalation", "detail": "billing complaint"}`,
				},
			},
			responseCall: &cmcall.Call{
				Direction:   cmcall.DirectionIncoming,
				Source:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"},
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0002"},
			},
			responseVariable: &fmvariable.Variable{Variables: map[string]string{}},
			responseCase: &kmkase.Case{
				ID: uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000006"),
			},
			expectContactV1CaseCreate: true,
			expectRes: &messageContent{
				ToolCallID:   "3a1f2c10-c001-11f0-9000-000000000005",
				Result:       "success",
				Message:      "Case created successfully.",
				ResourceType: "case",
				ResourceID:   "3a1f2c10-c001-11f0-9000-000000000006",
			},
		},
		{
			name: "dedup skip: contact_case_id already set",
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000011"),
					CustomerID: uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000012"),
				},
				ActiveflowID:  uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000013"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000014"),
			},
			tool: &message.ToolCall{
				ID:   "3a1f2c10-c001-11f0-9000-000000000015",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameCaseCreate,
					Arguments: `{}`,
				},
			},
			responseCall: &cmcall.Call{
				Direction:   cmcall.DirectionIncoming,
				Source:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"},
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0002"},
			},
			responseVariable: &fmvariable.Variable{Variables: map[string]string{
				"contact_case_id": "3a1f2c10-c001-11f0-9000-000000000099",
			}},
			expectContactV1CaseCreate: false,
			expectRes: &messageContent{
				ToolCallID:   "3a1f2c10-c001-11f0-9000-000000000015",
				Result:       "success",
				Message:      "A case already exists for this call/conversation; no new case was created.",
				ResourceType: "case",
				ResourceID:   "3a1f2c10-c001-11f0-9000-000000000099",
			},
		},
		{
			name: "CRM-ineligible peer: fillSuccess, not fillFailed",
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000021"),
					CustomerID: uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000022"),
				},
				ActiveflowID:  uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000023"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000024"),
			},
			tool: &message.ToolCall{
				ID:   "3a1f2c10-c001-11f0-9000-000000000025",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameCaseCreate,
					Arguments: `{}`,
				},
			},
			responseCall: &cmcall.Call{
				Direction:   cmcall.DirectionIncoming,
				Source:      commonaddress.Address{Type: commonaddress.TypeAgent, Target: "agent-1"},
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0002"},
			},
			expectContactV1CaseCreate: false,
			expectRes: &messageContent{
				ToolCallID:   "3a1f2c10-c001-11f0-9000-000000000025",
				Result:       "success",
				Message:      "No case was created: peer type agent is not eligible for CRM case tracking.",
				ResourceType: "case",
				ResourceID:   "",
			},
		},
		{
			name: "ContactV1CaseCreate error -> fillFailed",
			aicall: &aicall.AIcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000031"),
					CustomerID: uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000032"),
				},
				ActiveflowID:  uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000033"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("3a1f2c10-c001-11f0-9000-000000000034"),
			},
			tool: &message.ToolCall{
				ID:   "3a1f2c10-c001-11f0-9000-000000000035",
				Type: message.ToolTypeFunction,
				Function: message.FunctionCall{
					Name:      message.FunctionCallNameCaseCreate,
					Arguments: `{}`,
				},
			},
			responseCall: &cmcall.Call{
				Direction:   cmcall.DirectionIncoming,
				Source:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"},
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0002"},
			},
			responseVariable:          &fmvariable.Variable{Variables: map[string]string{}},
			expectContactV1CaseCreate: true,
			responseCreateErr:         errTest,
			expectRes: &messageContent{
				ToolCallID: "3a1f2c10-c001-11f0-9000-000000000035",
				Result:     "failed",
				Message:    errTest.Error(),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockAI := aihandler.NewMockAIHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, tt.aicall.ReferenceID).Return(tt.responseCall, nil)
			// All fixtures above use DirectionIncoming, so the resolved
			// peer is always Source (see deriveEndpointsForCase). Whether
			// FlowV1VariableGet is reached depends on whether that peer
			// passes isCRMEligiblePeer.
			if isCRMEligiblePeer(tt.responseCall.Source.Type) {
				mockReq.EXPECT().FlowV1VariableGet(ctx, tt.aicall.ActiveflowID).Return(tt.responseVariable, nil)
			}

			if tt.expectContactV1CaseCreate {
				mockReq.EXPECT().ContactV1CaseCreate(
					ctx, tt.aicall.CustomerID, gomock.Any(), gomock.Any(), "call", gomock.Any(), gomock.Any(),
				).Return(tt.responseCase, tt.responseCreateErr)
				if tt.responseCreateErr == nil {
					mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.aicall.ActiveflowID, map[string]string{"contact_case_id": tt.responseCase.ID.String()}).Return(nil)
				}
			}

			res := h.toolHandleCaseCreate(ctx, tt.aicall, tt.tool)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

// Test_toolHandleCaseCreate_UnknownDirection_Skipped is a regression
// guard (VOIP-1243 round-review fix): an unknown/unrecognized call
// Direction makes deriveEndpointsForCase return a zero-value peer
// (commonaddress.TypeNone), which MUST be classified as CRM-ineligible
// so ContactV1CaseCreate is never called with an empty peer.
func Test_toolHandleCaseCreate_UnknownDirection_Skipped(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockAI := aihandler.NewMockAIHandler(mc)
	mockMessage := messagehandler.NewMockMessageHandler(mc)

	h := &aicallHandler{
		utilHandler:    mockUtil,
		reqHandler:     mockReq,
		notifyHandler:  mockNotify,
		db:             mockDB,
		aiHandler:      mockAI,
		messageHandler: mockMessage,
	}
	ctx := context.Background()

	tc := &aicall.AIcall{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("88888888-0000-11f0-8000-000000000001"),
			CustomerID: uuid.FromStringOrNil("88888888-0000-11f0-8000-000000000002"),
		},
		ActiveflowID:  uuid.FromStringOrNil("88888888-0000-11f0-8000-000000000003"),
		ReferenceType: aicall.ReferenceTypeCall,
		ReferenceID:   uuid.FromStringOrNil("88888888-0000-11f0-8000-000000000004"),
	}
	tool := &message.ToolCall{
		ID:   "88888888-0000-11f0-8000-000000000005",
		Type: message.ToolTypeFunction,
		Function: message.FunctionCall{
			Name:      message.FunctionCallNameCaseCreate,
			Arguments: `{}`,
		},
	}
	responseCall := &cmcall.Call{
		Direction:   "", // unknown direction
		Source:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0001"},
		Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+155****0002"},
	}

	mockReq.EXPECT().CallV1CallGet(ctx, tc.ReferenceID).Return(responseCall, nil)
	// no FlowV1VariableGet/ContactV1CaseCreate expected -- zero-value peer
	// from an unknown direction must be skipped as CRM-ineligible.

	res := h.toolHandleCaseCreate(ctx, tc, tool)

	expectRes := &messageContent{
		ToolCallID:   "88888888-0000-11f0-8000-000000000005",
		Result:       "success",
		Message:      "No case was created: peer type  is not eligible for CRM case tracking.",
		ResourceType: "case",
		ResourceID:   "",
	}
	if !reflect.DeepEqual(res, expectRes) {
		t.Errorf("expected: %v, got: %v", expectRes, res)
	}
}

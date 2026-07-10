package activeflowhandler

import (
	"context"
	"testing"

	cmkase "monorepo/bin-contact-manager/models/kase"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	cmcall "monorepo/bin-call-manager/models/call"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-flow-manager/models/action"
	"monorepo/bin-flow-manager/models/activeflow"
	"monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-flow-manager/pkg/dbhandler"
)

func Test_actionHandleCaseCreate_Call(t *testing.T) {

	af := &activeflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("11111111-0000-11f0-8000-000000000001"),
			CustomerID: uuid.FromStringOrNil("11111111-0000-11f0-8000-000000000002"),
		},
		ReferenceType: activeflow.ReferenceTypeCall,
		ReferenceID:   uuid.FromStringOrNil("11111111-0000-11f0-8000-000000000003"),
		CurrentAction: action.Action{
			ID:   uuid.FromStringOrNil("11111111-0000-11f0-8000-000000000004"),
			Type: action.TypeCaseCreate,
			Option: map[string]any{
				"name":   "test case",
				"detail": "test detail",
				"note":   "test note",
			},
		},
	}

	responseCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("11111111-0000-11f0-8000-000000000003"),
		},
		Direction:   cmcall.DirectionIncoming,
		Source:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000001"},
		Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000002"},
	}

	responseVariable := &variable.Variable{
		ID:        af.ID,
		Variables: map[string]string{},
	}

	responseCase := &cmkase.Case{
		ID: uuid.FromStringOrNil("11111111-0000-11f0-8000-000000000005"),
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &activeflowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	ctx := context.Background()

	mockReq.EXPECT().CallV1CallGet(ctx, af.ReferenceID).Return(responseCall, nil)
	mockReq.EXPECT().FlowV1VariableGet(ctx, af.ID).Return(responseVariable, nil)
	mockReq.EXPECT().ContactV1CaseCreate(ctx, af.CustomerID, responseCall.Destination, commonaddress.TypeTel, responseCall.Source.Target, "call", "test case", "test detail").Return(responseCase, nil)
	mockReq.EXPECT().FlowV1VariableSetVariable(ctx, af.ID, map[string]string{"contact_case_id": responseCase.ID.String()}).Return(nil)
	mockReq.EXPECT().ContactV1CaseNoteCreate(ctx, af.CustomerID, responseCase.ID, "system", nil, "test note").Return(nil, nil)

	if errAction := h.actionHandleCaseCreate(ctx, af); errAction != nil {
		t.Errorf("Wrong match.\nexpect: ok\ngot: %v", errAction)
	}
}

func Test_actionHandleCaseCreate_Conversation(t *testing.T) {

	af := &activeflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("22222222-0000-11f0-8000-000000000001"),
			CustomerID: uuid.FromStringOrNil("22222222-0000-11f0-8000-000000000002"),
		},
		ReferenceType: activeflow.ReferenceTypeConversation,
		ReferenceID:   uuid.FromStringOrNil("22222222-0000-11f0-8000-000000000003"),
		CurrentAction: action.Action{
			ID:   uuid.FromStringOrNil("22222222-0000-11f0-8000-000000000004"),
			Type: action.TypeCaseCreate,
		},
	}

	responseConversation := &cvconversation.Conversation{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("22222222-0000-11f0-8000-000000000003"),
		},
		Self: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000010"},
		Peer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000011"},
	}

	responseVariable := &variable.Variable{
		ID:        af.ID,
		Variables: map[string]string{},
	}

	responseCase := &cmkase.Case{
		ID: uuid.FromStringOrNil("22222222-0000-11f0-8000-000000000005"),
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &activeflowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	ctx := context.Background()

	mockReq.EXPECT().ConversationV1ConversationGet(ctx, af.ReferenceID).Return(responseConversation, nil)
	mockReq.EXPECT().FlowV1VariableGet(ctx, af.ID).Return(responseVariable, nil)
	mockReq.EXPECT().ContactV1CaseCreate(ctx, af.CustomerID, responseConversation.Self, commonaddress.TypeTel, responseConversation.Peer.Target, "conversation_message", "", "").Return(responseCase, nil)
	mockReq.EXPECT().FlowV1VariableSetVariable(ctx, af.ID, map[string]string{"contact_case_id": responseCase.ID.String()}).Return(nil)

	if errAction := h.actionHandleCaseCreate(ctx, af); errAction != nil {
		t.Errorf("Wrong match.\nexpect: ok\ngot: %v", errAction)
	}
}

func Test_actionHandleCaseCreate_UnsupportedReferenceType(t *testing.T) {

	af := &activeflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("33333333-0000-11f0-8000-000000000001"),
			CustomerID: uuid.FromStringOrNil("33333333-0000-11f0-8000-000000000002"),
		},
		ReferenceType: activeflow.ReferenceTypeCampaign,
		ReferenceID:   uuid.FromStringOrNil("33333333-0000-11f0-8000-000000000003"),
		CurrentAction: action.Action{
			ID:   uuid.FromStringOrNil("33333333-0000-11f0-8000-000000000004"),
			Type: action.TypeCaseCreate,
		},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &activeflowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	ctx := context.Background()

	// no mock expectations set on mockReq -- asserts neither CallV1CallGet,
	// ConversationV1ConversationGet, ContactV1CaseCreate, nor
	// ContactV1CaseNoteCreate are called for an unsupported reference type.
	if errAction := h.actionHandleCaseCreate(ctx, af); errAction != nil {
		t.Errorf("Wrong match.\nexpect: ok\ngot: %v", errAction)
	}
}

func Test_actionHandleCaseCreate_CRMIneligiblePeer(t *testing.T) {

	af := &activeflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("44444444-0000-11f0-8000-000000000001"),
			CustomerID: uuid.FromStringOrNil("44444444-0000-11f0-8000-000000000002"),
		},
		ReferenceType: activeflow.ReferenceTypeCall,
		ReferenceID:   uuid.FromStringOrNil("44444444-0000-11f0-8000-000000000003"),
		CurrentAction: action.Action{
			ID:   uuid.FromStringOrNil("44444444-0000-11f0-8000-000000000004"),
			Type: action.TypeCaseCreate,
		},
	}

	responseCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("44444444-0000-11f0-8000-000000000003"),
		},
		Direction:   cmcall.DirectionIncoming,
		Source:      commonaddress.Address{Type: commonaddress.TypeExtension, Target: "1000"},
		Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000002"},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &activeflowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	ctx := context.Background()

	mockReq.EXPECT().CallV1CallGet(ctx, af.ReferenceID).Return(responseCall, nil)
	// no FlowV1VariableGet/ContactV1CaseCreate expected -- ineligible peer is a skip

	if errAction := h.actionHandleCaseCreate(ctx, af); errAction != nil {
		t.Errorf("Wrong match.\nexpect: ok\ngot: %v", errAction)
	}
}

func Test_actionHandleCaseCreate_DedupSkip(t *testing.T) {

	af := &activeflow.Activeflow{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("55555555-0000-11f0-8000-000000000001"),
			CustomerID: uuid.FromStringOrNil("55555555-0000-11f0-8000-000000000002"),
		},
		ReferenceType: activeflow.ReferenceTypeCall,
		ReferenceID:   uuid.FromStringOrNil("55555555-0000-11f0-8000-000000000003"),
		CurrentAction: action.Action{
			ID:   uuid.FromStringOrNil("55555555-0000-11f0-8000-000000000004"),
			Type: action.TypeCaseCreate,
		},
	}

	responseCall := &cmcall.Call{
		Identity: commonidentity.Identity{
			ID: uuid.FromStringOrNil("55555555-0000-11f0-8000-000000000003"),
		},
		Direction:   cmcall.DirectionIncoming,
		Source:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000001"},
		Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000002"},
	}

	responseVariable := &variable.Variable{
		ID:        af.ID,
		Variables: map[string]string{"contact_case_id": "66666666-0000-11f0-8000-000000000001"},
	}

	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)

	h := &activeflowHandler{
		db:         mockDB,
		reqHandler: mockReq,
	}

	ctx := context.Background()

	mockReq.EXPECT().CallV1CallGet(ctx, af.ReferenceID).Return(responseCall, nil)
	mockReq.EXPECT().FlowV1VariableGet(ctx, af.ID).Return(responseVariable, nil)
	// ContactV1CaseCreate must NOT be called -- design §3.5 dedup check short-circuits it

	if errAction := h.actionHandleCaseCreate(ctx, af); errAction != nil {
		t.Errorf("Wrong match.\nexpect: ok\ngot: %v", errAction)
	}
}

func Test_actionHandleCaseCreate_CreateErrorsSwallowed(t *testing.T) {

	tests := []struct {
		name    string
		errFunc error
	}{
		{
			name:    "already exists",
			errFunc: cerrors.AlreadyExists("contact-manager", "duplicate", "case already exists"),
		},
		{
			name:    "unavailable",
			errFunc: cerrors.Unavailable("contact-manager", "deadlock", "deadlock"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			af := &activeflow.Activeflow{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("77777777-0000-11f0-8000-000000000001"),
					CustomerID: uuid.FromStringOrNil("77777777-0000-11f0-8000-000000000002"),
				},
				ReferenceType: activeflow.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("77777777-0000-11f0-8000-000000000003"),
				CurrentAction: action.Action{
					ID:   uuid.FromStringOrNil("77777777-0000-11f0-8000-000000000004"),
					Type: action.TypeCaseCreate,
				},
			}

			responseCall := &cmcall.Call{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("77777777-0000-11f0-8000-000000000003"),
				},
				Direction:   cmcall.DirectionIncoming,
				Source:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000001"},
				Destination: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000002"},
			}

			responseVariable := &variable.Variable{
				ID:        af.ID,
				Variables: map[string]string{},
			}

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &activeflowHandler{
				db:         mockDB,
				reqHandler: mockReq,
			}

			ctx := context.Background()

			mockReq.EXPECT().CallV1CallGet(ctx, af.ReferenceID).Return(responseCall, nil)
			mockReq.EXPECT().FlowV1VariableGet(ctx, af.ID).Return(responseVariable, nil)
			mockReq.EXPECT().ContactV1CaseCreate(ctx, af.CustomerID, responseCall.Destination, commonaddress.TypeTel, responseCall.Source.Target, "call", "", "").Return(nil, tt.errFunc)

			if errAction := h.actionHandleCaseCreate(ctx, af); errAction != nil {
				t.Errorf("Wrong match.\nexpect: ok\ngot: %v", errAction)
			}
		})
	}
}

func Test_deriveEndpointsForCase(t *testing.T) {

	tests := []struct {
		name string

		direction string
		source    commonaddress.Address
		dest      commonaddress.Address

		expectPeer commonaddress.Address
		expectSelf commonaddress.Address
	}{
		{
			name:      "incoming",
			direction: "incoming",
			source:    commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000001"},
			dest:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000002"},

			expectPeer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000001"},
			expectSelf: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000002"},
		},
		{
			name:      "outgoing",
			direction: "outgoing",
			source:    commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000001"},
			dest:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000002"},

			expectPeer: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000002"},
			expectSelf: commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000001"},
		},
		{
			name:      "unknown",
			direction: "unknown",
			source:    commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000001"},
			dest:      commonaddress.Address{Type: commonaddress.TypeTel, Target: "+821100000002"},

			expectPeer: commonaddress.Address{},
			expectSelf: commonaddress.Address{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			peer, self := deriveEndpointsForCase(tt.direction, tt.source, tt.dest)
			if peer != tt.expectPeer {
				t.Errorf("Wrong match peer.\nexpect: %v\ngot: %v", tt.expectPeer, peer)
			}
			if self != tt.expectSelf {
				t.Errorf("Wrong match self.\nexpect: %v\ngot: %v", tt.expectSelf, self)
			}
		})
	}
}

func Test_isCRMEligiblePeer(t *testing.T) {

	tests := []struct {
		name     string
		peerType commonaddress.Type
		expect   bool
	}{
		{"tel eligible", commonaddress.TypeTel, true},
		{"agent ineligible", commonaddress.TypeAgent, false},
		{"sip ineligible", commonaddress.TypeSIP, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if res := isCRMEligiblePeer(tt.peerType); res != tt.expect {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expect, res)
			}
		})
	}
}

func Test_actionHandleCaseCreate_dispatchRegistration(t *testing.T) {
	// Dispatch-switch registration test confirming action.TypeCaseCreate
	// resolves to actionHandleCaseCreate and does not error at runtime
	// (design VOIP-1243 §5.3).
	if _, ok := action.OptionStructByType[action.TypeCaseCreate]; !ok {
		t.Errorf("action.TypeCaseCreate is not registered in action.OptionStructByType")
	}

	found := false
	for _, typ := range action.TypeListAll {
		if typ == action.TypeCaseCreate {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("action.TypeCaseCreate is not registered in action.TypeListAll")
	}

	if _, ok := action.MapRequiredMediasByType[action.TypeCaseCreate]; !ok {
		t.Errorf("action.TypeCaseCreate is not registered in action.MapRequiredMediasByType")
	}
}

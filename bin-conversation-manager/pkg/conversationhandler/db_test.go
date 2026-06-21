package conversationhandler

import (
	"context"
	"reflect"
	"strings"
	"testing"

	amagent "monorepo/bin-agent-manager/models/agent"
	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	stderrors "errors"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
)

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseConversation *conversation.Conversation
	}{
		{
			"normal",

			uuid.FromStringOrNil("e0258e08-e6e8-11ec-b5c7-ff2400334630"),

			&conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("e0258e08-e6e8-11ec-b5c7-ff2400334630"),
					CustomerID: uuid.FromStringOrNil("31fb223a-e6e7-11ec-9e22-438ecfd00508"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &conversationHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConversationGet(ctx, tt.id).Return(tt.responseConversation, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConversation) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseConversation, res)
			}
		})
	}
}

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID       uuid.UUID
		conversationName string
		detail           string
		referenceType    conversation.Type
		referenceID      string
		self             commonaddress.Address
		peer             commonaddress.Address

		responseUUID         uuid.UUID
		responseConversation *conversation.Conversation

		expectConversation *conversation.Conversation
	}{
		{
			name: "normal",

			customerID:       uuid.FromStringOrNil("31fb223a-e6e7-11ec-9e22-438ecfd00508"),
			conversationName: "test conversation",
			detail:           "test detail",
			referenceType:    conversation.TypeLine,
			referenceID:      "3dc385f8-e6e7-11ec-9250-5f6c3097570f",
			self: commonaddress.Address{
				Type:   commonaddress.TypeLine,
				Target: "2fcb542c-f113-11ec-a7de-6335ee489d7b",
			},
			peer: commonaddress.Address{
				Type:       commonaddress.TypeLine,
				Target:     "46bc98c0-e6e7-11ec-a93f-479cd0ec28a9",
				TargetName: "test participant",
			},

			responseUUID: uuid.FromStringOrNil("d2a852d8-0069-11ee-96b8-3fffef7f1833"),
			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("1c73620a-e6e8-11ec-89d7-a788fc793ba3"),
					CustomerID: uuid.FromStringOrNil("31fb223a-e6e7-11ec-9e22-438ecfd00508"),
				},
			},

			expectConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d2a852d8-0069-11ee-96b8-3fffef7f1833"),
					CustomerID: uuid.FromStringOrNil("31fb223a-e6e7-11ec-9e22-438ecfd00508"),
				},
				Name:     "test conversation",
				Detail:   "test detail",
				Type:     conversation.TypeLine,
				DialogID: "3dc385f8-e6e7-11ec-9250-5f6c3097570f",
				Self: commonaddress.Address{
					Type:   commonaddress.TypeLine,
					Target: "2fcb542c-f113-11ec-a7de-6335ee489d7b",
				},
				Peer: commonaddress.Address{
					Type:       commonaddress.TypeLine,
					Target:     "46bc98c0-e6e7-11ec-a93f-479cd0ec28a9",
					TargetName: "test participant",
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &conversationHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)

			mockDB.EXPECT().ConversationCreate(ctx, tt.expectConversation).Return(nil)
			mockDB.EXPECT().ConversationGet(ctx, gomock.Any()).Return(tt.responseConversation, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseConversation.CustomerID, conversation.EventTypeConversationCreated, tt.responseConversation)

			_, err := h.Create(ctx, tt.customerID, tt.conversationName, tt.detail, tt.referenceType, tt.referenceID, tt.self, tt.peer)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[conversation.Field]any

		responseConversations []*conversation.Conversation
	}{
		{
			name: "normal",

			pageToken: "2022-04-18T03:22:17.995000Z",
			pageSize:  100,
			filters: map[conversation.Field]any{
				conversation.FieldCustomerID: uuid.FromStringOrNil("62fe906c-3e13-11ef-9a64-270aea3013c5"),
				conversation.FieldDeleted:    false,
			},

			responseConversations: []*conversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("643d8d88-e862-11ec-a93c-bf31836c63e8"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &conversationHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}
			ctx := context.Background()

			mockDB.EXPECT().ConversationList(ctx, tt.pageSize, tt.pageToken, tt.filters).Return(tt.responseConversations, nil)

			res, err := h.List(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConversations) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseConversations, res)
			}

		})
	}
}

func Test_Update(t *testing.T) {

	tests := []struct {
		name string

		id     uuid.UUID
		fields map[conversation.Field]any

		// existingConversation is what h.Get returns during the pre-fetch when
		// owner_id is non-nil (used to derive cv.CustomerID for the
		// same-customer agent constraint). Set to nil for cases that do not
		// trigger the pre-fetch (no owner_id, or owner_id == uuid.Nil).
		existingConversation *conversation.Conversation
		// agentGetResponse is what AgentV1AgentGet returns when invoked.
		// Set to nil for cases that do not trigger an agent get call.
		agentGetResponse *amagent.Agent

		expectFields         map[conversation.Field]any
		responseConversation *conversation.Conversation
	}{
		{
			name: "no owner_id present — fields untouched",

			id: uuid.FromStringOrNil("4455607e-006a-11ee-bfbb-032b6e5d2c44"),
			fields: map[conversation.Field]any{
				conversation.FieldName: "update name",
			},

			expectFields: map[conversation.Field]any{
				conversation.FieldName: "update name",
			},
			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4455607e-006a-11ee-bfbb-032b6e5d2c44"),
				},
			},
		},
		{
			name: "owner_id non-nil with valid agent — derives owner_type=agent",

			id: uuid.FromStringOrNil("17a8d3a4-2604-11f0-9a7d-eb1f4d6f9a01"),
			fields: map[conversation.Field]any{
				conversation.FieldOwnerID: uuid.FromStringOrNil("2c4a4c2a-2604-11f0-aa18-3b3f1b8a1b22"),
			},

			existingConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("17a8d3a4-2604-11f0-9a7d-eb1f4d6f9a01"),
					CustomerID: uuid.FromStringOrNil("3a3a3a3a-2604-11f0-9a7d-eb1f4d6f9a01"),
				},
			},
			agentGetResponse: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2c4a4c2a-2604-11f0-aa18-3b3f1b8a1b22"),
					CustomerID: uuid.FromStringOrNil("3a3a3a3a-2604-11f0-9a7d-eb1f4d6f9a01"),
				},
			},

			expectFields: map[conversation.Field]any{
				conversation.FieldOwnerID:   uuid.FromStringOrNil("2c4a4c2a-2604-11f0-aa18-3b3f1b8a1b22"),
				conversation.FieldOwnerType: commonidentity.OwnerTypeAgent,
			},
			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("17a8d3a4-2604-11f0-9a7d-eb1f4d6f9a01"),
				},
			},
		},
		{
			name: "owner_id nil — derives owner_type=\"\" (unassign), no agent validation",

			id: uuid.FromStringOrNil("3f4d6c00-2604-11f0-bf9a-cf1a4d2f9c33"),
			fields: map[conversation.Field]any{
				conversation.FieldOwnerID: uuid.Nil,
			},

			expectFields: map[conversation.Field]any{
				conversation.FieldOwnerID:   uuid.Nil,
				conversation.FieldOwnerType: commonidentity.OwnerTypeNone,
			},
			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("3f4d6c00-2604-11f0-bf9a-cf1a4d2f9c33"),
				},
			},
		},
		{
			name: "owner_type from caller is overridden by derived value",

			id: uuid.FromStringOrNil("57a91b46-2604-11f0-9a3d-d3a8a45f9d44"),
			fields: map[conversation.Field]any{
				conversation.FieldOwnerID:   uuid.FromStringOrNil("6b2a3d80-2604-11f0-a4f6-c3b8a4ad2e55"),
				conversation.FieldOwnerType: commonidentity.OwnerType("something-else"),
			},

			existingConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("57a91b46-2604-11f0-9a3d-d3a8a45f9d44"),
					CustomerID: uuid.FromStringOrNil("8c2a3d80-2604-11f0-a4f6-c3b8a4ad2e55"),
				},
			},
			agentGetResponse: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("6b2a3d80-2604-11f0-a4f6-c3b8a4ad2e55"),
					CustomerID: uuid.FromStringOrNil("8c2a3d80-2604-11f0-a4f6-c3b8a4ad2e55"),
				},
			},

			expectFields: map[conversation.Field]any{
				conversation.FieldOwnerID:   uuid.FromStringOrNil("6b2a3d80-2604-11f0-a4f6-c3b8a4ad2e55"),
				conversation.FieldOwnerType: commonidentity.OwnerTypeAgent,
			},
			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("57a91b46-2604-11f0-9a3d-d3a8a45f9d44"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &conversationHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
				reqHandler:    mockReq,
			}
			ctx := context.Background()

			// If the case triggers the pre-fetch (owner_id non-nil), expect
			// h.Get -> ConversationGet to load the existing conversation, and
			// expect AgentV1AgentGet for per-id agent lookup.
			if tt.existingConversation != nil {
				mockDB.EXPECT().ConversationGet(ctx, tt.id).Return(tt.existingConversation, nil)

				ownerID := tt.fields[conversation.FieldOwnerID].(uuid.UUID)
				mockReq.EXPECT().AgentV1AgentGet(ctx, ownerID).Return(tt.agentGetResponse, nil)
			}

			mockDB.EXPECT().ConversationUpdate(ctx, tt.id, tt.expectFields).Return(nil)
			mockDB.EXPECT().ConversationGet(ctx, tt.id).Return(tt.responseConversation, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseConversation.CustomerID, conversation.EventTypeConversationUpdated, tt.responseConversation)

			res, err := h.Update(ctx, tt.id, tt.fields)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConversation) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseConversation, res)
			}
		})
	}
}

// Test_Update_AgentNotFound covers the canonical not-found case (and any
// transport error from AgentV1AgentGet, which agent-manager today collapses
// into HTTP 500 over RPC). Both surface as InvalidArgument/AGENT_NOT_FOUND
// per design §5.4.
func Test_Update_AgentNotFound(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockLine := linehandler.NewMockLineHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &conversationHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		lineHandler:   mockLine,
		reqHandler:    mockReq,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("a3b4c5d6-2604-11f0-9a7d-eb1f4d6f9a01")
	ownerID := uuid.FromStringOrNil("b3b4c5d6-2604-11f0-9a7d-eb1f4d6f9a01")
	customerID := uuid.FromStringOrNil("c3b4c5d6-2604-11f0-9a7d-eb1f4d6f9a01")
	fields := map[conversation.Field]any{
		conversation.FieldOwnerID: ownerID,
	}

	existing := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: customerID,
		},
	}

	getErr := stderrors.New("agent not found")
	mockDB.EXPECT().ConversationGet(ctx, id).Return(existing, nil)
	mockReq.EXPECT().AgentV1AgentGet(ctx, ownerID).Return(nil, getErr)
	// No DB write, no post-update get, no event publish.

	res, err := h.Update(ctx, id, fields)
	if err == nil {
		t.Fatalf("expected error, got nil (res=%v)", res)
	}
	if res != nil {
		t.Errorf("expected nil result on error, got: %v", res)
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("expected *cerrors.VoipbinError, got: %T (%v)", err, err)
	}
	if ve.Status != cerrors.StatusInvalidArgument {
		t.Errorf("expected Status=InvalidArgument, got: %v", ve.Status)
	}
	if ve.Reason != "AGENT_NOT_FOUND" {
		t.Errorf("expected Reason=AGENT_NOT_FOUND, got: %v", ve.Reason)
	}
	if !stderrors.Is(err, getErr) {
		t.Errorf("expected wrapped getErr in chain, got: %v", err)
	}
}

// Test_Update_AgentCustomerMismatch covers the case where the agent exists
// but belongs to a different customer than the conversation. This must
// surface a distinct typed error so the api-manager edge can show users
// which constraint failed (design §5.4).
func Test_Update_AgentCustomerMismatch(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockLine := linehandler.NewMockLineHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &conversationHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		lineHandler:   mockLine,
		reqHandler:    mockReq,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("11aa22bb-2604-11f0-9a7d-eb1f4d6f9a01")
	ownerID := uuid.FromStringOrNil("22bb33cc-2604-11f0-9a7d-eb1f4d6f9a01")
	conversationCustomerID := uuid.FromStringOrNil("33cc44dd-2604-11f0-9a7d-eb1f4d6f9a01")
	agentCustomerID := uuid.FromStringOrNil("44dd55ee-2604-11f0-9a7d-eb1f4d6f9a01")
	fields := map[conversation.Field]any{
		conversation.FieldOwnerID: ownerID,
	}

	existing := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         id,
			CustomerID: conversationCustomerID,
		},
	}
	agentResp := &amagent.Agent{
		Identity: commonidentity.Identity{
			ID:         ownerID,
			CustomerID: agentCustomerID, // distinct from the conversation's customer
		},
	}

	mockDB.EXPECT().ConversationGet(ctx, id).Return(existing, nil)
	mockReq.EXPECT().AgentV1AgentGet(ctx, ownerID).Return(agentResp, nil)
	// No DB write, no post-update get, no event publish.

	res, err := h.Update(ctx, id, fields)
	if err == nil {
		t.Fatalf("expected error, got nil (res=%v)", res)
	}
	if res != nil {
		t.Errorf("expected nil result on error, got: %v", res)
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("expected *cerrors.VoipbinError, got: %T (%v)", err, err)
	}
	if ve.Status != cerrors.StatusInvalidArgument {
		t.Errorf("expected Status=InvalidArgument, got: %v", ve.Status)
	}
	if ve.Reason != "AGENT_CUSTOMER_MISMATCH" {
		t.Errorf("expected Reason=AGENT_CUSTOMER_MISMATCH, got: %v", ve.Reason)
	}
	// Defense-in-depth: the user-facing message must NOT include the cross-tenant
	// agent_customer_id or the redundant conversation_customer_id. Operators can
	// recover those from server-side structured logs.
	if strings.Contains(ve.Message, "agent_customer_id") {
		t.Errorf("agent_customer_id leaked into user-facing message: %q", ve.Message)
	}
	if strings.Contains(ve.Message, "conversation_customer_id") {
		t.Errorf("conversation_customer_id leaked into user-facing message: %q", ve.Message)
	}
}

func Test_Update_ConversationGetFailure(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockLine := linehandler.NewMockLineHandler(mc)
	mockReq := requesthandler.NewMockRequestHandler(mc)
	h := &conversationHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		lineHandler:   mockLine,
		reqHandler:    mockReq,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("11223344-2604-11f0-9a7d-eb1f4d6f9a01")
	ownerID := uuid.FromStringOrNil("22334455-2604-11f0-9a7d-eb1f4d6f9a01")
	fields := map[conversation.Field]any{
		conversation.FieldOwnerID: ownerID,
	}

	getErr := stderrors.New("database read failed")
	mockDB.EXPECT().ConversationGet(ctx, id).Return(nil, getErr)
	// No agent list call, no DB write, no event publish.

	res, err := h.Update(ctx, id, fields)
	if err == nil {
		t.Fatalf("expected error, got nil (res=%v)", res)
	}
	if res != nil {
		t.Errorf("expected nil result on error, got: %v", res)
	}
	if !stderrors.Is(err, getErr) {
		t.Errorf("expected wrapped getErr in chain, got: %v", err)
	}
}

func Test_Update_invalidOwnerIDType(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	mockLine := linehandler.NewMockLineHandler(mc)
	h := &conversationHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
		lineHandler:   mockLine,
	}
	ctx := context.Background()

	id := uuid.FromStringOrNil("a3a91b46-2604-11f0-bb1c-3f7d8a3aef66")
	fields := map[conversation.Field]any{
		// Caller passed a string instead of uuid.UUID — defensive type-assertion
		// failure path; ConvertStringMapToFieldMap should normally produce
		// uuid.UUID typed values, but we guard against malformed callers.
		conversation.FieldOwnerID: "not-a-uuid",
	}

	// No DB calls should happen — derivation rejects the request before the
	// DB write.
	res, err := h.Update(ctx, id, fields)
	if err == nil {
		t.Fatalf("expected error, got nil (res=%v)", res)
	}
	if res != nil {
		t.Errorf("expected nil result on error, got: %v", res)
	}

	var ve *cerrors.VoipbinError
	if !stderrors.As(err, &ve) {
		t.Fatalf("expected *cerrors.VoipbinError, got: %T (%v)", err, err)
	}
	if ve.Status != cerrors.StatusInvalidArgument {
		t.Errorf("expected Status=InvalidArgument, got: %v", ve.Status)
	}
}

func Test_GetOrCreateBySelfAndPeer_Existing(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &conversationHandler{
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001")
	conversationType := conversation.TypeMessage
	dialogID := ""
	self := commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: "+1234567890",
	}
	peer := commonaddress.Address{
		Type:       commonaddress.TypeTel,
		Target:     "+0987654321",
		TargetName: "Peer Name",
	}

	expectedConv := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			CustomerID: customerID,
		},
		Type:     conversationType,
		DialogID: dialogID,
		Self:     self,
		Peer:     peer,
	}

	mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, self, peer).Return(expectedConv, nil)

	res, err := h.GetOrCreateBySelfAndPeer(ctx, customerID, conversationType, dialogID, self, peer)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if res.ID != expectedConv.ID {
		t.Errorf("Wrong ID. expect: %s, got: %s", expectedConv.ID, res.ID)
	}
}

func Test_GetOrCreateBySelfAndPeer_Create(t *testing.T) {
	mc := gomock.NewController(t)
	defer mc.Finish()

	mockUtil := utilhandler.NewMockUtilHandler(mc)
	mockDB := dbhandler.NewMockDBHandler(mc)
	mockNotify := notifyhandler.NewMockNotifyHandler(mc)
	h := &conversationHandler{
		utilHandler:   mockUtil,
		db:            mockDB,
		notifyHandler: mockNotify,
	}
	ctx := context.Background()

	customerID := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001")
	conversationType := conversation.TypeMessage
	dialogID := ""
	self := commonaddress.Address{
		Type:   commonaddress.TypeTel,
		Target: "+1234567890",
	}
	peer := commonaddress.Address{
		Type:       commonaddress.TypeTel,
		Target:     "+0987654321",
		TargetName: "Peer Name",
	}

	newID := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440003")
	createdConv := &conversation.Conversation{
		Identity: commonidentity.Identity{
			ID:         newID,
			CustomerID: customerID,
		},
		Owner: commonidentity.Owner{
			OwnerType: commonidentity.OwnerTypeNone,
			OwnerID:   uuid.Nil,
		},
		Name:   "SMS · Peer Name (+0987654321)",
		Detail: "SMS conversation",
		Type:     conversationType,
		DialogID: dialogID,
		Self:     self,
		Peer:     peer,
	}

	mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, self, peer).Return(nil, dbhandler.ErrNotFound)
	mockUtil.EXPECT().UUIDCreate().Return(newID)
	mockDB.EXPECT().ConversationCreate(ctx, createdConv).Return(nil)
	mockDB.EXPECT().ConversationGet(ctx, newID).Return(createdConv, nil)
	mockNotify.EXPECT().PublishWebhookEvent(ctx, customerID, conversation.EventTypeConversationCreated, createdConv)

	res, err := h.GetOrCreateBySelfAndPeer(ctx, customerID, conversationType, dialogID, self, peer)
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}

	if res.ID != newID {
		t.Errorf("Wrong ID. expect: %s, got: %s", newID, res.ID)
	}
}

// Test_GetOrCreateBySelfAndPeer_NormalizesSelfPeer is a regression test proving
// that GetOrCreateBySelfAndPeer canonicalizes self.Target/peer.Target via
// commonaddress.NormalizeTarget BEFORE the dedup lookup
// (ConversationGetBySelfAndPeer). A punctuated tel target must reach the DB in
// its canonical '+'-prefixed digit-only form so a caller's formatting variant
// hits the same stored conversation. A whatsapp waID with no '+' must normalize
// to itself (digit-only, idempotent, no '+' injected). DialogID is NOT
// normalized and must pass through untouched.
func Test_GetOrCreateBySelfAndPeer_NormalizesSelfPeer(t *testing.T) {
	tests := []struct {
		name string

		dialogID string
		self     commonaddress.Address
		peer     commonaddress.Address

		expectSelf commonaddress.Address
		expectPeer commonaddress.Address
	}{
		{
			name: "punctuated tel self and peer are canonicalized",

			dialogID: "dialog-keep-me-untouched",
			self: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: " +1 (650) 555-7890 ",
			},
			peer: commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+1-202-555-0123",
				TargetName: "Peer Name",
			},

			expectSelf: commonaddress.Address{
				Type:   commonaddress.TypeTel,
				Target: "+16505557890",
			},
			expectPeer: commonaddress.Address{
				Type:       commonaddress.TypeTel,
				Target:     "+12025550123",
				TargetName: "Peer Name",
			},
		},
		{
			name: "whatsapp waID without '+' normalizes to itself (idempotent, no '+' injected)",

			dialogID: "wa-dialog-id",
			self: commonaddress.Address{
				Type:   commonaddress.TypeWhatsApp,
				Target: "15551234567",
			},
			peer: commonaddress.Address{
				Type:   commonaddress.TypeWhatsApp,
				Target: "15559876543",
			},

			expectSelf: commonaddress.Address{
				Type:   commonaddress.TypeWhatsApp,
				Target: "15551234567",
			},
			expectPeer: commonaddress.Address{
				Type:   commonaddress.TypeWhatsApp,
				Target: "15559876543",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			h := &conversationHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
			}
			ctx := context.Background()

			customerID := uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001")
			conversationType := conversation.TypeMessage

			expectedConv := &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440099"),
					CustomerID: customerID,
				},
				Type:     conversationType,
				DialogID: tt.dialogID,
				Self:     tt.expectSelf,
				Peer:     tt.expectPeer,
			}

			// Assert the dedup lookup receives the CANONICAL self/peer.
			matchSelf := gomock.Cond(func(x any) bool {
				a, ok := x.(commonaddress.Address)
				return ok && reflect.DeepEqual(a, tt.expectSelf)
			})
			matchPeer := gomock.Cond(func(x any) bool {
				a, ok := x.(commonaddress.Address)
				return ok && reflect.DeepEqual(a, tt.expectPeer)
			})
			mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, matchSelf, matchPeer).Return(expectedConv, nil)

			res, err := h.GetOrCreateBySelfAndPeer(ctx, customerID, conversationType, tt.dialogID, tt.self, tt.peer)
			if err != nil {
				t.Errorf("Expected no error, got: %v", err)
			}

			if res.ID != expectedConv.ID {
				t.Errorf("Wrong ID. expect: %s, got: %s", expectedConv.ID, res.ID)
			}

			// DialogID must pass through untouched (NOT normalized).
			if res.DialogID != tt.dialogID {
				t.Errorf("DialogID changed. expect: %q, got: %q", tt.dialogID, res.DialogID)
			}
		})
	}
}

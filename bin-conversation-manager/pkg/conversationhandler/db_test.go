package conversationhandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	cerrors "monorepo/bin-common-handler/models/errors"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
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
			name: "owner_id non-nil — derives owner_type=agent",

			id: uuid.FromStringOrNil("17a8d3a4-2604-11f0-9a7d-eb1f4d6f9a01"),
			fields: map[conversation.Field]any{
				conversation.FieldOwnerID: uuid.FromStringOrNil("2c4a4c2a-2604-11f0-aa18-3b3f1b8a1b22"),
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
			name: "owner_id nil — derives owner_type=\"\" (unassign)",

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
			h := &conversationHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}
			ctx := context.Background()

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
		Name:     "conversation with Peer Name",
		Detail:   "conversation with Peer Name",
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

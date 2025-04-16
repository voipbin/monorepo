package conversationhandler

import (
	"context"
	"reflect"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

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

func Test_GetByReferenceInfo(t *testing.T) {

	tests := []struct {
		name string

		conversationType conversation.Type
		dialogID         string

		responseConversation *conversation.Conversation
	}{
		{
			name: "normal",

			conversationType: conversation.TypeLine,
			dialogID:         "a481fe6c-e6e9-11ec-92f7-6366decbd9e8",

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("a9341b0c-e6e9-11ec-a3a2-0b511930bae5"),
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

			mockDB.EXPECT().ConversationGetByTypeAndDialogID(ctx, tt.conversationType, tt.dialogID).Return(tt.responseConversation, nil)

			res, err := h.GetByTypeAndDialogID(ctx, tt.conversationType, tt.dialogID)
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

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[string]string

		responseConversations []*conversation.Conversation
	}{
		{
			name: "normal",

			pageToken: "2022-04-18 03:22:17.995000",
			pageSize:  100,
			filters: map[string]string{
				"customer_id": "62fe906c-3e13-11ef-9a64-270aea3013c5",
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

			mockDB.EXPECT().ConversationGets(ctx, tt.pageSize, tt.pageToken, tt.filters).Return(tt.responseConversations, nil)

			res, err := h.Gets(ctx, tt.pageToken, tt.pageSize, tt.filters)
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

		id               uuid.UUID
		conversationName string
		detail           string

		responseConversation *conversation.Conversation
	}{
		{
			name: "normal",

			id:               uuid.FromStringOrNil("4455607e-006a-11ee-bfbb-032b6e5d2c44"),
			conversationName: "test name",
			detail:           "test detail",

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("4455607e-006a-11ee-bfbb-032b6e5d2c44"),
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

			mockDB.EXPECT().ConversationSet(ctx, tt.id, tt.conversationName, tt.detail).Return(nil)
			mockDB.EXPECT().ConversationGet(ctx, tt.id).Return(tt.responseConversation, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseConversation.CustomerID, conversation.EventTypeConversationUpdated, tt.responseConversation)

			res, err := h.Update(ctx, tt.id, tt.conversationName, tt.detail)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseConversation) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseConversation, res)
			}
		})
	}
}

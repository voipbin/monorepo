package messagehandler

import (
	"context"
	reflect "reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID     uuid.UUID
		conversationID uuid.UUID
		direction      message.Direction
		status         message.Status
		referenceType  message.ReferenceType
		referenceID    uuid.UUID
		transactionID  string
		text           string
		medias         []media.Media

		responseUUID    uuid.UUID
		responseMessage *message.Message

		expectMessage *message.Message
	}{
		{
			name: "normal",

			customerID:     uuid.FromStringOrNil("8db56df0-e86e-11ec-b6d7-1fa3ca565837"),
			conversationID: uuid.FromStringOrNil("8e0e1dce-e86e-11ec-9537-77df0d80af26"),
			direction:      message.DirectionIncoming,
			status:         message.StatusDone,
			referenceType:  message.ReferenceTypeLine,
			referenceID:    uuid.FromStringOrNil("59946c7c-f1d5-11ec-bdad-2323294b508e"),
			transactionID:  "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
			text:           "Hello world",
			medias:         []media.Media{},

			responseUUID: uuid.FromStringOrNil("f6834112-0240-11ee-8146-2fb17ae9ef0a"),
			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("03b6a572-e870-11ec-a4a7-0bf92e5d8985"),
					CustomerID: uuid.FromStringOrNil("8db56df0-e86e-11ec-b6d7-1fa3ca565837"),
				},
			},

			expectMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f6834112-0240-11ee-8146-2fb17ae9ef0a"),
					CustomerID: uuid.FromStringOrNil("8db56df0-e86e-11ec-b6d7-1fa3ca565837"),
				},
				ConversationID: uuid.FromStringOrNil("8e0e1dce-e86e-11ec-9537-77df0d80af26"),
				Direction:      message.DirectionIncoming,
				Status:         message.StatusDone,
				ReferenceType:  message.ReferenceTypeLine,
				ReferenceID:    uuid.FromStringOrNil("59946c7c-f1d5-11ec-bdad-2323294b508e"),
				TransactionID:  "Ud871bcaf7c3ad13d2a0b0d78a42a287f",
				Text:           "Hello world",
				Medias:         []media.Media{},
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
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &messageHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.responseUUID).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessage.CustomerID, message.EventTypeMessageCreated, tt.responseMessage)
			res, err := h.Create(ctx, uuid.Nil, tt.customerID, tt.conversationID, tt.direction, tt.status, tt.referenceType, tt.referenceID, tt.transactionID, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseMessage, res)
			}

		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseMessage *message.Message
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("af7c3310-f1d8-11ec-a2f1-db31b43cade8"),

			responseMessage: &message.Message{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("af7c3310-f1d8-11ec-a2f1-db31b43cade8"),
					CustomerID: uuid.FromStringOrNil("8db56df0-e86e-11ec-b6d7-1fa3ca565837"),
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
			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}
			ctx := context.Background()

			mockDB.EXPECT().MessageDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseMessage.CustomerID, message.EventTypeMessageDeleted, tt.responseMessage)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseMessage, res)
			}
		})
	}
}

func Test_List(t *testing.T) {

	tests := []struct {
		name string

		pageToken string
		pageSize  uint64
		filters   map[message.Field]any

		responseMessages []*message.Message
	}{
		{
			name: "normal",

			pageToken: "2022-04-18 03:22:17.995000",
			pageSize:  100,
			filters: map[message.Field]any{
				message.FieldCustomerID: uuid.FromStringOrNil("853f266e-1bda-11f0-b63c-43ea1644089b"),
			},

			responseMessages: []*message.Message{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("a6b8ac2a-1bda-11f0-a905-cbf684282511"),
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
			h := &messageHandler{
				db:            mockDB,
				notifyHandler: mockNotify,
				lineHandler:   mockLine,
			}
			ctx := context.Background()

			mockDB.EXPECT().MessageList(ctx, tt.pageToken, tt.pageSize, tt.filters).Return(tt.responseMessages, nil)

			res, err := h.List(ctx, tt.pageToken, tt.pageSize, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessages) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.responseMessages, res)
			}
		})
	}
}

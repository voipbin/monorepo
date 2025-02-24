package messagehandler

import (
	"context"
	"monorepo/bin-chatbot-manager/models/message"
	"monorepo/bin-chatbot-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID    uuid.UUID
		chatbotcallID uuid.UUID
		direction     message.Direction
		role          message.Role
		content       string

		responseUUID uuid.UUID

		expectMessage *message.Message
	}{
		{
			name: "have all",

			customerID:    uuid.FromStringOrNil("f227397c-f260-11ef-b217-4f6ff6930cf2"),
			chatbotcallID: uuid.FromStringOrNil("f26fd614-f260-11ef-ae2f-ab1a2508e20d"),
			direction:     message.DirectionIncoming,
			role:          message.RoleUser,
			content:       "Hello, world!",

			responseUUID: uuid.FromStringOrNil("f2a827da-f260-11ef-9766-8b270f0b8d97"),

			expectMessage: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("f2a827da-f260-11ef-9766-8b270f0b8d97"),
					CustomerID: uuid.FromStringOrNil("f227397c-f260-11ef-b217-4f6ff6930cf2"),
				},
				ChatbotcallID: uuid.FromStringOrNil("f26fd614-f260-11ef-ae2f-ab1a2508e20d"),

				Direction: message.DirectionIncoming,
				Role:      message.RoleUser,
				Content:   "Hello, world!",
			},
		},
		{
			name: "empty",

			responseUUID: uuid.FromStringOrNil("0812955a-f262-11ef-a3a2-1bee273dee65"),

			expectMessage: &message.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("0812955a-f262-11ef-a3a2-1bee273dee65"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockDB := dbhandler.NewMockDBHandler(mc)
			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)

			h := messageHandler{
				utilHandler:   mockUtil,
				db:            mockDB,
				notifyHandler: mockNotify,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.responseUUID).Return(tt.expectMessage, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage.CustomerID, message.EventTypeMessageCreated, tt.expectMessage)

			res, err := h.Create(ctx, tt.customerID, tt.chatbotcallID, tt.direction, tt.role, tt.content)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectMessage, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		chatbotcallID uuid.UUID
		size          uint64
		token         string
		filters       map[string]string

		responseMessages []*message.Message
	}{
		{
			name: "normal",

			chatbotcallID: uuid.FromStringOrNil("5774f2dc-f262-11ef-b704-bb967f775316"),
			size:          10,
			token:         "2023-01-03 21:35:02.809",
			filters: map[string]string{
				"deleted": "false",
			},

			responseMessages: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("57bb3986-f262-11ef-b6db-57288b2a39c3"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &messageHandler{
				utilHandler:   mockUtil,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessageGets(ctx, tt.chatbotcallID, tt.size, tt.token, tt.filters).Return(tt.responseMessages, nil)

			res, err := h.Gets(ctx, tt.chatbotcallID, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessages) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseMessages, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseMessage *message.Message
	}{
		{
			name: "normal",

			id: uuid.FromStringOrNil("2f6f4928-f2c0-11ef-b7ce-fbb7241790f5"),

			responseMessage: &message.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("2f6f4928-f2c0-11ef-b7ce-fbb7241790f5"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &messageHandler{
				utilHandler:   mockUtil,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessageGet(ctx, tt.id).Return(tt.responseMessage, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseMessage, res)
			}
		})
	}
}

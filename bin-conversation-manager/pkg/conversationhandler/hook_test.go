package conversationhandler

import (
	"context"
	"testing"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/messagehandler"
)

func Test_Hook(t *testing.T) {

	tests := []struct {
		name string

		uri  string
		data []byte

		expectAccountID uuid.UUID

		responseAccount       *account.Account
		responseConversations []*conversation.Conversation
		responseUUIDs         []uuid.UUID
		responseMessages      []*message.Message
	}{
		{
			name: "normal",

			uri: "https://hook.voipbin.net/v1.0/conversation/accounts/e8f5795a-e6eb-11ec-bb81-c3cec34bd99c",
			data: []byte(`{
				"destination": "U11298214116e3afbad432b5794a6d3a0",
				"events": [
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "16173792131295",
							"text": "Hello"
						},
						"webhookEventId": "01G49KHTWA1D2WF05D0VHEMGZE",
						"deliveryContext": {
							"isRedelivery": false
						},
						"timestamp": 1653884906096,
						"source": {
							"type": "user",
							"userId": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
						},
						"replyToken": "4bdd674a22cc479b8e9e429465396b76",
						"mode": "active"
					}
				]
			}`),

			expectAccountID: uuid.FromStringOrNil("e8f5795a-e6eb-11ec-bb81-c3cec34bd99c"),

			responseAccount: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e8f5795a-e6eb-11ec-bb81-c3cec34bd99c"),
				},
				Type: account.TypeLine,
			},
			responseConversations: []*conversation.Conversation{
				{
					Identity: commonidentity.Identity{
						CustomerID: uuid.FromStringOrNil("e8f5795a-e6eb-11ec-bb81-c3cec34bd99c"),
					},
				},
			},
			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("cb285f42-0075-11ee-ad73-0fae8c027ffc"),
			},
			responseMessages: []*message.Message{
				{},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mocKUtil := utilhandler.NewMockUtilHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockAccount := accounthandler.NewMockAccountHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &conversationHandler{
				utilHandler:    mocKUtil,
				db:             mockDB,
				notifyHandler:  mockNotify,
				accountHandler: mockAccount,
				messageHandler: mockMessage,
				lineHandler:    mockLine,
			}
			ctx := context.Background()

			mockAccount.EXPECT().Get(ctx, tt.expectAccountID).Return(tt.responseAccount, nil)

			mockLine.EXPECT().Hook(ctx, tt.responseAccount, tt.data).Return(tt.responseConversations, tt.responseMessages, nil)

			// conversations
			for i := 0; i < len(tt.responseConversations); i++ {
				mocKUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDs[i])
				mockDB.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().ConversationGet(ctx, gomock.Any()).Return(&conversation.Conversation{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), conversation.EventTypeConversationCreated, gomock.Any())
			}

			// messages
			for range tt.responseMessages {
				mockLine.EXPECT().GetParticipant(ctx, gomock.Any(), gomock.Any()).Return(&commonaddress.Address{}, nil)
				mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, gomock.Any(), gomock.Any()).Return(&conversation.Conversation{}, nil)

				mockMessage.EXPECT().Create(
					ctx,
					gomock.Any(),
					gomock.Any(),
					message.DirectionIncoming,
					message.StatusDone,
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
					gomock.Any(),
				).Return(&message.Message{}, nil)
			}

			if err := h.Hook(ctx, tt.uri, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_hookLine(t *testing.T) {

	tests := []struct {
		name string

		account *account.Account
		data    []byte

		responseConversation *conversation.Conversation

		responseUUIDs         []uuid.UUID
		responseConversations []*conversation.Conversation
		responseMessages      []*message.Message
	}{
		{
			name: "normal",

			account: &account.Account{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("e9eb2682-e6ed-11ec-a8f2-0b533280b1ae"),
				},
			},
			data: []byte(`{
				"destination": "U11298214116e3afbad432b5794a6d3a0",
				"events": [
					{
						"type": "message",
						"message": {
							"type": "text",
							"id": "16173792131295",
							"text": "Hello"
						},
						"webhookEventId": "01G49KHTWA1D2WF05D0VHEMGZE",
						"deliveryContext": {
							"isRedelivery": false
						},
						"timestamp": 1653884906096,
						"source": {
							"type": "user",
							"userId": "Ud871bcaf7c3ad13d2a0b0d78a42a287f"
						},
						"replyToken": "4bdd674a22cc479b8e9e429465396b76",
						"mode": "active"
					}
				]
			}`),

			responseConversation: &conversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("f7f25d6c-e874-11ec-b140-3f088b887f43"),
				},
				ReferenceType: conversation.ReferenceTypeLine,
			},

			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("4d94b5ca-0076-11ee-8c59-6fff2eb90055"),
			},
			responseConversations: []*conversation.Conversation{
				{
					Identity: commonidentity.Identity{
						CustomerID: uuid.FromStringOrNil("e8f5795a-e6eb-11ec-bb81-c3cec34bd99c"),
					},
				},
			},
			responseMessages: []*message.Message{
				{},
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
			mockMessage := messagehandler.NewMockMessageHandler(mc)
			mockLine := linehandler.NewMockLineHandler(mc)
			h := &conversationHandler{
				utilHandler:    mockUtil,
				db:             mockDB,
				notifyHandler:  mockNotify,
				messageHandler: mockMessage,
				lineHandler:    mockLine,
			}

			ctx := context.Background()

			mockLine.EXPECT().Hook(ctx, tt.account, tt.data).Return(tt.responseConversations, tt.responseMessages, nil)

			// conversations
			for i := range tt.responseConversations {
				mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDs[i])
				mockDB.EXPECT().ConversationCreate(ctx, gomock.Any()).Return(nil)
				mockDB.EXPECT().ConversationGet(ctx, gomock.Any()).Return(&conversation.Conversation{}, nil)
				mockNotify.EXPECT().PublishWebhookEvent(ctx, gomock.Any(), conversation.EventTypeConversationCreated, gomock.Any())
			}

			// messages
			for _, tmp := range tt.responseMessages {

				mockLine.EXPECT().GetParticipant(ctx, gomock.Any(), gomock.Any()).Return(&commonaddress.Address{}, nil)
				mockDB.EXPECT().ConversationGetBySelfAndPeer(ctx, gomock.Any(), gomock.Any()).Return(tt.responseConversation, nil)

				mockMessage.EXPECT().Create(
					ctx,
					tt.responseConversation.CustomerID,
					tt.responseConversation.ID,
					message.DirectionIncoming,
					message.StatusDone,
					tt.responseConversation.ReferenceType,
					tt.responseConversation.ReferenceID,
					"",
					tmp.Text,
					nil,
				).Return(&message.Message{}, nil)
			}

			if err := h.hookLine(ctx, tt.account, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

package conversationhandler

import (
	"context"
	"testing"

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

		responseAccount  *account.Account
		responseUUIDs    []uuid.UUID
		responseMessages []*message.Message
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
			mockLine.EXPECT().Hook(ctx, tt.responseAccount, tt.data).Return(nil)

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

		responseUUIDs    []uuid.UUID
		responseMessages []*message.Message
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
				Type: conversation.TypeLine,
			},

			responseUUIDs: []uuid.UUID{
				uuid.FromStringOrNil("4d94b5ca-0076-11ee-8c59-6fff2eb90055"),
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

			mockLine.EXPECT().Hook(ctx, tt.account, tt.data).Return(nil)

			if err := h.hookLine(ctx, tt.account, tt.data); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

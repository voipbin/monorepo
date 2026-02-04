package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmedia "monorepo/bin-conversation-manager/models/media"
	cvmessage "monorepo/bin-conversation-manager/models/message"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ConversationMessageListByConversationID(t *testing.T) {

	tests := []struct {
		name string

		customer       *amagent.Agent
		conversationID uuid.UUID
		pageToken      string
		pageSize       uint64

		responseConversation *cvconversation.Conversation
		responseMessages     []cvmessage.Message

		expectFilters map[cvmessage.Field]any
		expectRes     []*cvmessage.WebhookMessage
	}{
		{
			name: "normal",

			customer: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			conversationID: uuid.FromStringOrNil("ee26103a-ed24-11ec-bfa1-7b247ecf7e93"),
			pageToken:      "2020-10-20T01:00:00.995000Z",
			pageSize:       10,

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("ee26103a-ed24-11ec-bfa1-7b247ecf7e93"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseMessages: []cvmessage.Message{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("13c78e5e-ed25-11ec-b924-b319c14e0209"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("13e8436a-ed25-11ec-ba44-8b0716e4b2f0"),
					},
				},
			},

			expectFilters: map[cvmessage.Field]any{
				cvmessage.FieldDeleted:        false,
				cvmessage.FieldConversationID: uuid.FromStringOrNil("ee26103a-ed24-11ec-bfa1-7b247ecf7e93"),
			},
			expectRes: []*cvmessage.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("13c78e5e-ed25-11ec-b924-b319c14e0209"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("13e8436a-ed25-11ec-ba44-8b0716e4b2f0"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			mockReq.EXPECT().ConversationV1MessageList(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.responseMessages, nil)

			res, err := h.ConversationMessageGetsByConversationID(ctx, tt.customer, tt.conversationID, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationMessageSend(t *testing.T) {

	tests := []struct {
		name string

		customer       *amagent.Agent
		conversationID uuid.UUID
		text           string
		medias         []cvmedia.Media

		responseConversation *cvconversation.Conversation
		responseMessage      *cvmessage.Message

		expectRes *cvmessage.WebhookMessage
	}{
		{
			name: "simple text message",

			customer: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			conversationID: uuid.FromStringOrNil("8dd8eda0-ed25-11ec-9b1a-07913127a65a"),
			text:           "hello world",
			medias:         []cvmedia.Media{},

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("8dd8eda0-ed25-11ec-9b1a-07913127a65a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseMessage: &cvmessage.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c9bd73a4-ed25-11ec-8283-43aafea65e87"),
				},
			},

			expectRes: &cvmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("c9bd73a4-ed25-11ec-8283-43aafea65e87"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.responseConversation, nil)
			mockReq.EXPECT().ConversationV1MessageSend(ctx, tt.conversationID, tt.text, tt.medias).Return(tt.responseMessage, nil)

			res, err := h.ConversationMessageSend(ctx, tt.customer, tt.conversationID, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

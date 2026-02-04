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

func Test_ServiceAgentConversationMessageList(t *testing.T) {

	type test struct {
		name string

		agent          *amagent.Agent
		conversationID uuid.UUID
		size           uint64
		token          string

		responseConversation *cvconversation.Conversation
		responseMessages     []cvmessage.Message

		expectFilters map[cvmessage.Field]any
		expectRes     []*cvmessage.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			conversationID: uuid.FromStringOrNil("d186a8c4-3ed3-11ef-8ff9-931b5d4f8461"),
			size:           100,
			token:          "2021-03-01T01:00:00.995000Z",

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d186a8c4-3ed3-11ef-8ff9-931b5d4f8461"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},
			responseMessages: []cvmessage.Message{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("25d8503a-3ed4-11ef-b8a8-2b447608d9c5"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("25febbda-3ed4-11ef-86de-272011cf8e77"),
					},
				},
			},

			expectFilters: map[cvmessage.Field]any{
				cvmessage.FieldDeleted:        false,
				cvmessage.FieldConversationID: uuid.FromStringOrNil("d186a8c4-3ed3-11ef-8ff9-931b5d4f8461"),
			},
			expectRes: []*cvmessage.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("25d8503a-3ed4-11ef-b8a8-2b447608d9c5"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("25febbda-3ed4-11ef-86de-272011cf8e77"),
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
			mockReq.EXPECT().ConversationV1MessageList(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseMessages, nil)

			res, err := h.ServiceAgentConversationMessageList(ctx, tt.agent, tt.conversationID, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentConversationMessageSend(t *testing.T) {

	type test struct {
		name string

		agent          *amagent.Agent
		conversationID uuid.UUID
		text           string
		medias         []cvmedia.Media

		responseConversation        *cvconversation.Conversation
		responseConversationMessage *cvmessage.Message

		expectRes *cvmessage.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
					CustomerID: uuid.FromStringOrNil("5d16712c-3b9f-11ef-8a51-f30f1e2ce1e9"),
				},
				Permission: amagent.PermissionCustomerAgent,
			},
			conversationID: uuid.FromStringOrNil("d186a8c4-3ed3-11ef-8ff9-931b5d4f8461"),
			text:           "test message",
			medias:         []cvmedia.Media{},

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("d186a8c4-3ed3-11ef-8ff9-931b5d4f8461"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},
			responseConversationMessage: &cvmessage.Message{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("25d8503a-3ed4-11ef-b8a8-2b447608d9c5"),
				},
			},

			expectRes: &cvmessage.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("25d8503a-3ed4-11ef-b8a8-2b447608d9c5"),
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
			mockReq.EXPECT().ConversationV1MessageSend(ctx, tt.conversationID, tt.text, tt.medias).Return(tt.responseConversationMessage, nil)

			res, err := h.ServiceAgentConversationMessageSend(ctx, tt.agent, tt.conversationID, tt.text, tt.medias)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

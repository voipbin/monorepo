package servicehandler

import (
	"context"
	amagent "monorepo/bin-agent-manager/models/agent"
	"monorepo/bin-api-manager/pkg/dbhandler"
	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_ServiceAgentConversationGet(t *testing.T) {

	type test struct {
		name string

		agent          *amagent.Agent
		conversationID uuid.UUID

		responseConversation *cvconversation.Conversation
		expectRes            *cvconversation.WebhookMessage
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
			conversationID: uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				},
			},
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("14189ed4-3ed1-11ef-8056-bffadb501e2f"),
				},
				Owner: commonidentity.Owner{
					OwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
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

			res, err := h.ServiceAgentConversationGet(ctx, tt.agent, tt.conversationID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_ServiceAgentConversationList(t *testing.T) {

	type test struct {
		name string

		agent *amagent.Agent
		size  uint64
		token string

		responseConversations []cvconversation.Conversation

		expectFilters map[cvconversation.Field]any
		expectRes     []*cvconversation.WebhookMessage
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
			size:  100,
			token: "2021-03-01 01:00:00.995000",

			responseConversations: []cvconversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("620bce9e-3ed2-11ef-b45a-3f6e2898153d"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("62a1ec8a-3ed2-11ef-bb8c-a788ea1ad2ad"),
					},
				},
			},

			expectFilters: map[cvconversation.Field]any{
				cvconversation.FieldOwnerID: uuid.FromStringOrNil("5cd8c836-3b9f-11ef-98ac-db226570f09a"),
				cvconversation.FieldDeleted: false,
			},
			expectRes: []*cvconversation.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("620bce9e-3ed2-11ef-b45a-3f6e2898153d"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("62a1ec8a-3ed2-11ef-bb8c-a788ea1ad2ad"),
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

			mockReq.EXPECT().ConversationV1ConversationList(ctx, tt.token, tt.size, tt.expectFilters).Return(tt.responseConversations, nil)

			res, err := h.ServiceAgentConversationList(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(tt.expectRes, res) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	cvconversation "monorepo/bin-conversation-manager/models/conversation"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ConversationGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name      string
		agent     *amagent.Agent
		pageToken string
		pageSize  uint64

		responseConversations []cvconversation.Conversation

		expectFilters map[cvconversation.Field]any
		expectRes     []*cvconversation.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},
			pageToken: "2020-10-20T01:00:00.995000",
			pageSize:  10,

			responseConversations: []cvconversation.Conversation{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("18965a18-ed21-11ec-89d2-b7e541377482"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("18c13288-ed21-11ec-9d0f-c7be55dc87d7"),
					},
				},
			},
			expectFilters: map[cvconversation.Field]any{
				cvconversation.FieldCustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				cvconversation.FieldDeleted:    false,
			},
			expectRes: []*cvconversation.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("18965a18-ed21-11ec-89d2-b7e541377482"),
					},
				},
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("18c13288-ed21-11ec-9d0f-c7be55dc87d7"),
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

			mockReq.EXPECT().ConversationV1ConversationGets(ctx, tt.pageToken, tt.pageSize, tt.expectFilters).Return(tt.responseConversations, nil)
			res, err := h.ConversationGetsByCustomerID(ctx, tt.agent, tt.pageSize, tt.pageToken)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(res, tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationGet(t *testing.T) {

	tests := []struct {
		name           string
		customer       *amagent.Agent
		conversationID uuid.UUID

		response  *cvconversation.Conversation
		expectRes *cvconversation.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			uuid.FromStringOrNil("828e75ba-ed24-11ec-bbf2-7f0e56ac76f1"),

			&cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("828e75ba-ed24-11ec-bbf2-7f0e56ac76f1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("828e75ba-ed24-11ec-bbf2-7f0e56ac76f1"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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

			mockReq.EXPECT().ConversationV1ConversationGet(ctx, tt.conversationID).Return(tt.response, nil)
			res, err := h.ConversationGet(ctx, tt.customer, tt.conversationID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ConversationUpdate(t *testing.T) {

	tests := []struct {
		name  string
		agent *amagent.Agent

		conversationID uuid.UUID
		fileds         map[cvconversation.Field]any

		responseConversation *cvconversation.Conversation
		expectRes            *cvconversation.WebhookMessage
	}{
		{
			name: "normal",
			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionCustomerAdmin,
			},

			conversationID: uuid.FromStringOrNil("50fbe844-007d-11ee-a616-0fe1db6961e5"),
			fileds: map[cvconversation.Field]any{
				cvconversation.FieldName:   "test name",
				cvconversation.FieldDetail: "test detail",
			},

			responseConversation: &cvconversation.Conversation{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("50fbe844-007d-11ee-a616-0fe1db6961e5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			expectRes: &cvconversation.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("50fbe844-007d-11ee-a616-0fe1db6961e5"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
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
			mockReq.EXPECT().ConversationV1ConversationUpdate(ctx, tt.conversationID, tt.fileds).Return(tt.responseConversation, nil)
			res, err := h.ConversationUpdate(ctx, tt.agent, tt.conversationID, tt.fileds)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if reflect.DeepEqual(*res, *tt.expectRes) != true {
				t.Errorf("Wrong match.\nexpect: %v\n, got: %v\n", tt.expectRes, res)
			}
		})
	}
}

package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	chatbotchatbotcall "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_ChatbotcallGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		size    uint64
		token   string
		filters map[string]string

		response  []chatbotchatbotcall.Chatbotcall
		expectRes []*chatbotchatbotcall.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			10,
			"2020-09-20 03:23:20.995000",
			map[string]string{
				"deleted": "false",
			},

			[]chatbotchatbotcall.Chatbotcall{
				{
					ID: uuid.FromStringOrNil("78b58aef-2fcf-4a88-81e2-054f4e4c37d4"),
				},
			},
			[]*chatbotchatbotcall.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("78b58aef-2fcf-4a88-81e2-054f4e4c37d4"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().ChatbotV1ChatbotcallGetsByCustomerID(ctx, tt.agent.CustomerID, tt.token, tt.size, tt.filters).Return(tt.response, nil)

			res, err := h.ChatbotcallGetsByCustomerID(ctx, tt.agent, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotcallGet(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		chatbotcallID uuid.UUID

		response  *chatbotchatbotcall.Chatbotcall
		expectRes *chatbotchatbotcall.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),

			&chatbotchatbotcall.Chatbotcall{
				ID:         uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&chatbotchatbotcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().ChatbotV1ChatbotcallGet(ctx, tt.chatbotcallID).Return(tt.response, nil)

			res, err := h.ChatbotcallGet(ctx, tt.agent, tt.chatbotcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotcallDelete(t *testing.T) {

	tests := []struct {
		name string

		agent         *amagent.Agent
		chatbotcallID uuid.UUID

		responseChat *chatbotchatbotcall.Chatbotcall
		expectRes    *chatbotchatbotcall.WebhookMessage
	}{
		{
			"normal",
			&amagent.Agent{
				ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				Permission: amagent.PermissionCustomerAdmin,
			},
			uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			&chatbotchatbotcall.Chatbotcall{
				ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
			&chatbotchatbotcall.WebhookMessage{
				ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
				CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().ChatbotV1ChatbotcallGet(ctx, tt.chatbotcallID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatbotV1ChatbotcallDelete(ctx, tt.chatbotcallID).Return(tt.responseChat, nil)

			res, err := h.ChatbotcallDelete(ctx, tt.agent, tt.chatbotcallID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

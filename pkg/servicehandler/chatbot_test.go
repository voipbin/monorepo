package servicehandler

import (
	"context"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"github.com/golang/mock/gomock"
	chatbotchatbot "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/api-manager.git/pkg/dbhandler"
)

func Test_ChatbotCreate(t *testing.T) {

	tests := []struct {
		name string

		customer    *cscustomer.Customer
		chatbotName string
		detail      string
		engineType  chatbotchatbot.EngineType
		initPrompt  string

		response  *chatbotchatbot.Chatbot
		expectRes *chatbotchatbot.WebhookMessage
	}{
		{
			name: "normal",

			customer: &cscustomer.Customer{
				ID: uuid.FromStringOrNil("f0d20d08-376f-11ed-9a7a-dbc21a700b21"),
			},
			chatbotName: "test name",
			detail:      "test detail",
			engineType:  chatbotchatbot.EngineTypeChatGPT,
			initPrompt:  "test init prompt",

			response: &chatbotchatbot.Chatbot{
				ID: uuid.FromStringOrNil("ea4b81a9-ffab-4c20-8a77-c9e4d80df548"),
			},
			expectRes: &chatbotchatbot.WebhookMessage{
				ID: uuid.FromStringOrNil("ea4b81a9-ffab-4c20-8a77-c9e4d80df548"),
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

			mockReq.EXPECT().ChatbotV1ChatbotCreate(ctx, tt.customer.ID, tt.chatbotName, tt.detail, tt.engineType, tt.initPrompt).Return(tt.response, nil)

			res, err := h.ChatbotCreate(ctx, tt.customer, tt.chatbotName, tt.detail, tt.engineType, tt.initPrompt)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(*res, *tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", *tt.expectRes, *res)
			}
		})
	}
}

func Test_ChatbotGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		customer *cscustomer.Customer
		size     uint64
		token    string

		response  []chatbotchatbot.Chatbot
		expectRes []*chatbotchatbot.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("040422b6-3771-11ed-801b-27518c703c82"),
			},
			10,
			"2020-09-20 03:23:20.995000",

			[]chatbotchatbot.Chatbot{
				{
					ID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
				},
			},
			[]*chatbotchatbot.WebhookMessage{
				{
					ID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
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

			mockReq.EXPECT().ChatbotV1ChatbotGetsByCustomerID(ctx, tt.customer.ID, tt.token, tt.size).Return(tt.response, nil)

			res, err := h.ChatbotGetsByCustomerID(ctx, tt.customer, tt.size, tt.token)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotGet(t *testing.T) {

	tests := []struct {
		name string

		customer  *cscustomer.Customer
		chatbotID uuid.UUID

		response  *chatbotchatbot.Chatbot
		expectRes *chatbotchatbot.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("539c78aa-3771-11ed-ab19-379f45ca7efc"),
			},
			uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),

			&chatbotchatbot.Chatbot{
				ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
				CustomerID: uuid.FromStringOrNil("539c78aa-3771-11ed-ab19-379f45ca7efc"),
			},
			&chatbotchatbot.WebhookMessage{
				ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
				CustomerID: uuid.FromStringOrNil("539c78aa-3771-11ed-ab19-379f45ca7efc"),
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

			mockReq.EXPECT().ChatbotV1ChatbotGet(ctx, tt.chatbotID).Return(tt.response, nil)

			res, err := h.ChatbotGet(ctx, tt.customer, tt.chatbotID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect:%v\ngot:%v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotDelete(t *testing.T) {

	tests := []struct {
		name string

		customer  *cscustomer.Customer
		chatbotID uuid.UUID

		responseChat *chatbotchatbot.Chatbot
		expectRes    *chatbotchatbot.WebhookMessage
	}{
		{
			"normal",
			&cscustomer.Customer{
				ID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			&chatbotchatbot.Chatbot{
				ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
			},
			&chatbotchatbot.WebhookMessage{
				ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
				CustomerID: uuid.FromStringOrNil("852b9d5e-7ff9-11ec-9ca0-cf3c47e8c96b"),
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

			mockReq.EXPECT().ChatbotV1ChatbotGet(ctx, tt.chatbotID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatbotV1ChatbotDelete(ctx, tt.chatbotID).Return(tt.responseChat, nil)

			res, err := h.ChatbotDelete(ctx, tt.customer, tt.chatbotID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

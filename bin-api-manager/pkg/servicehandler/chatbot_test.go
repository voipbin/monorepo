package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"

	chatbotchatbot "monorepo/bin-chatbot-manager/models/chatbot"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ChatbotCreate(t *testing.T) {

	tests := []struct {
		name string

		agent               *amagent.Agent
		chatbotName         string
		detail              string
		engineType          chatbotchatbot.EngineType
		engineModel         chatbotchatbot.EngineModel
		initPrompt          string
		credentialBase64    string
		credentialProjectID string

		response  *chatbotchatbot.Chatbot
		expectRes *chatbotchatbot.WebhookMessage
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
			chatbotName:         "test name",
			detail:              "test detail",
			engineType:          chatbotchatbot.EngineTypeChatGPT,
			engineModel:         chatbotchatbot.EngineModelChatGPT4,
			initPrompt:          "test init prompt",
			credentialBase64:    "test credential base64",
			credentialProjectID: "test credential project id",

			response: &chatbotchatbot.Chatbot{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ea4b81a9-ffab-4c20-8a77-c9e4d80df548"),
				},
			},
			expectRes: &chatbotchatbot.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("ea4b81a9-ffab-4c20-8a77-c9e4d80df548"),
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

			mockReq.EXPECT().ChatbotV1ChatbotCreate(ctx, tt.agent.CustomerID, tt.chatbotName, tt.detail, tt.engineType, tt.engineModel, tt.initPrompt, tt.credentialBase64, tt.credentialProjectID).Return(tt.response, nil)

			res, err := h.ChatbotCreate(ctx, tt.agent, tt.chatbotName, tt.detail, tt.engineType, tt.engineModel, tt.initPrompt, tt.credentialBase64, tt.credentialProjectID)
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

		agent   *amagent.Agent
		size    uint64
		token   string
		filters map[string]string

		response  []chatbotchatbot.Chatbot
		expectRes []*chatbotchatbot.WebhookMessage
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
			10,
			"2020-09-20 03:23:20.995000",
			map[string]string{
				"deleted": "false",
			},

			[]chatbotchatbot.Chatbot{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
					},
				},
			},
			[]*chatbotchatbot.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("1dacd73f-5dca-46bd-b408-d703409ec557"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ChatbotV1ChatbotGetsByCustomerID(ctx, tt.agent.CustomerID, tt.token, tt.size, tt.filters).Return(tt.response, nil)

			res, err := h.ChatbotGetsByCustomerID(ctx, tt.agent, tt.size, tt.token)
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

		agent     *amagent.Agent
		chatbotID uuid.UUID

		response  *chatbotchatbot.Chatbot
		expectRes *chatbotchatbot.WebhookMessage
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
			uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),

			&chatbotchatbot.Chatbot{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&chatbotchatbot.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("90c9bd58-0cb0-4e7a-b55a-cef9f1570b63"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}
			ctx := context.Background()

			mockReq.EXPECT().ChatbotV1ChatbotGet(ctx, tt.chatbotID).Return(tt.response, nil)

			res, err := h.ChatbotGet(ctx, tt.agent, tt.chatbotID)
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

		agent     *amagent.Agent
		chatbotID uuid.UUID

		responseChat *chatbotchatbot.Chatbot
		expectRes    *chatbotchatbot.WebhookMessage
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
			uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),

			&chatbotchatbot.Chatbot{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&chatbotchatbot.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
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
			h := serviceHandler{
				reqHandler: mockReq,
				dbHandler:  mockDB,
			}

			ctx := context.Background()

			mockReq.EXPECT().ChatbotV1ChatbotGet(ctx, tt.chatbotID).Return(tt.responseChat, nil)
			mockReq.EXPECT().ChatbotV1ChatbotDelete(ctx, tt.chatbotID).Return(tt.responseChat, nil)

			res, err := h.ChatbotDelete(ctx, tt.agent, tt.chatbotID)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

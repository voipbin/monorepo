package servicehandler

import (
	"context"
	"reflect"
	"testing"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cbchatbot "monorepo/bin-chatbot-manager/models/chatbot"
	cbchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"

	amagent "monorepo/bin-agent-manager/models/agent"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"

	"monorepo/bin-api-manager/pkg/dbhandler"
)

func Test_ChatbotcallCreate(t *testing.T) {

	type test struct {
		name string

		agent         *amagent.Agent
		chatbotID     uuid.UUID
		referenceType cbchatbotcall.ReferenceType
		referenceID   uuid.UUID
		gender        cbchatbotcall.Gender
		language      string

		responseChatbot     *cbchatbot.Chatbot
		responseChatbotcall *cbchatbotcall.Chatbotcall

		expectRes *cbchatbotcall.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			chatbotID:     uuid.FromStringOrNil("3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46"),
			referenceType: cbchatbotcall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("f201d402-4596-47cf-87b9-bc6d234d286a"),
			gender:        cbchatbotcall.GenderMale,
			language:      "en-US",

			responseChatbot: &cbchatbot.Chatbot{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("3fc2c1b0-efaa-11ef-84bb-a7e8fba38e46"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			responseChatbotcall: &cbchatbotcall.Chatbotcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("407e793c-efaa-11ef-b0f4-4bdbcd626589"),
				},
			},

			expectRes: &cbchatbotcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("407e793c-efaa-11ef-b0f4-4bdbcd626589"),
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().ChatbotV1ChatbotGet(ctx, tt.chatbotID).Return(tt.responseChatbot, nil)
			mockReq.EXPECT().ChatbotV1ChatbotcallStart(ctx, tt.chatbotID, tt.referenceType, tt.referenceID, tt.gender, tt.language).Return(tt.responseChatbotcall, nil)

			res, err := h.ChatbotcallCreate(ctx, tt.agent, tt.chatbotID, tt.referenceType, tt.referenceID, tt.gender, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

func Test_ChatbotcallGetsByCustomerID(t *testing.T) {

	tests := []struct {
		name string

		agent   *amagent.Agent
		size    uint64
		token   string
		filters map[string]string

		response  []cbchatbotcall.Chatbotcall
		expectRes []*cbchatbotcall.WebhookMessage
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

			[]cbchatbotcall.Chatbotcall{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("78b58aef-2fcf-4a88-81e2-054f4e4c37d4"),
					},
				},
			},
			[]*cbchatbotcall.WebhookMessage{
				{
					Identity: commonidentity.Identity{
						ID: uuid.FromStringOrNil("78b58aef-2fcf-4a88-81e2-054f4e4c37d4"),
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

		response  *cbchatbotcall.Chatbotcall
		expectRes *cbchatbotcall.WebhookMessage
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
			uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),

			&cbchatbotcall.Chatbotcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cbchatbotcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("2c10c2af-fb73-416e-ab86-8e91e7db32c4"),
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

		responseChat *cbchatbotcall.Chatbotcall
		expectRes    *cbchatbotcall.WebhookMessage
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

			&cbchatbotcall.Chatbotcall{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
			},
			&cbchatbotcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("b35fcdb7-f3ee-4534-b6fa-24d78b437356"),
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

func Test_ChatbotcallSendMessage(t *testing.T) {

	type test struct {
		name string

		agent         *amagent.Agent
		chatbotcallID uuid.UUID
		role          cbchatbotcall.MessageRole
		text          string

		responseChatbotcall *cbchatbotcall.Chatbotcall

		expectRes *cbchatbotcall.WebhookMessage
	}

	tests := []test{
		{
			name: "normal",

			agent: &amagent.Agent{
				Identity: commonidentity.Identity{
					ID:         uuid.FromStringOrNil("d152e69e-105b-11ee-b395-eb18426de979"),
					CustomerID: uuid.FromStringOrNil("5f621078-8e5f-11ee-97b2-cfe7337b701c"),
				},
				Permission: amagent.PermissionProjectSuperAdmin,
			},
			chatbotcallID: uuid.FromStringOrNil("2a8154e6-efab-11ef-b017-4380cdc88e94"),
			role:          cbchatbotcall.MessageRoleUser,
			text:          "Hello, World!",

			responseChatbotcall: &cbchatbotcall.Chatbotcall{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a8154e6-efab-11ef-b017-4380cdc88e94"),
				},
			},

			expectRes: &cbchatbotcall.WebhookMessage{
				Identity: commonidentity.Identity{
					ID: uuid.FromStringOrNil("2a8154e6-efab-11ef-b017-4380cdc88e94"),
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
			mockUtil := utilhandler.NewMockUtilHandler(mc)

			h := serviceHandler{
				reqHandler:  mockReq,
				dbHandler:   mockDB,
				utilHandler: mockUtil,
			}
			ctx := context.Background()

			mockReq.EXPECT().ChatbotV1ChatbotcallGet(ctx, tt.chatbotcallID).Return(tt.responseChatbotcall, nil)
			mockReq.EXPECT().ChatbotV1ChatbotcallSendMessage(ctx, tt.chatbotcallID, tt.role, tt.text, 30000).Return(tt.responseChatbotcall, nil)

			res, err := h.ChatbotcallSendMessage(ctx, tt.agent, tt.chatbotcallID, tt.role, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v\n", tt.expectRes, res)
			}
		})
	}
}

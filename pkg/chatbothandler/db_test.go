package chatbothandler

import (
	"context"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/dbhandler"
)

func Test_Create(t *testing.T) {

	tests := []struct {
		name string

		customerID  uuid.UUID
		chatbotName string
		detail      string
		engineType  chatbot.EngineType
		initPrompt  string

		responseUUID    uuid.UUID
		responseChatbot *chatbot.Chatbot

		expectChatbot *chatbot.Chatbot
	}{
		{
			name: "normal",

			customerID:  uuid.FromStringOrNil("8db73654-a70d-11ed-ae15-6726993338d8"),
			chatbotName: "test name",
			detail:      "test detail",
			engineType:  chatbot.EngineTypeChatGPT,
			initPrompt:  "test init prompt",

			responseUUID: uuid.FromStringOrNil("8dedbf26-a70d-11ed-be65-3ba04faa629b"),
			responseChatbot: &chatbot.Chatbot{
				ID: uuid.FromStringOrNil("8dedbf26-a70d-11ed-be65-3ba04faa629b"),
			},

			expectChatbot: &chatbot.Chatbot{
				ID:         uuid.FromStringOrNil("8dedbf26-a70d-11ed-be65-3ba04faa629b"),
				CustomerID: uuid.FromStringOrNil("8db73654-a70d-11ed-ae15-6726993338d8"),
				Name:       "test name",
				Detail:     "test detail",
				EngineType: chatbot.EngineTypeChatGPT,
				InitPrompt: "test init prompt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &chatbotHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID)
			mockDB.EXPECT().ChatbotCreate(ctx, tt.expectChatbot).Return(nil)
			mockDB.EXPECT().ChatbotGet(ctx, tt.responseUUID).Return(tt.responseChatbot, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbot.CustomerID, chatbot.EventTypeChatbotCreated, tt.responseChatbot)

			res, err := h.Create(ctx, tt.customerID, tt.chatbotName, tt.detail, tt.engineType, tt.initPrompt)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbot) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbot, res)
			}
		})
	}
}

func Test_Gets(t *testing.T) {

	tests := []struct {
		name string

		customerID uuid.UUID
		size       uint64
		token      string
		filters    map[string]string

		responseChatbots []*chatbot.Chatbot
	}{
		{
			name: "normal",

			customerID: uuid.FromStringOrNil("132be434-f839-11ed-ae95-efa657af10fb"),
			size:       10,
			token:      "2023-01-03 21:35:02.809",
			filters: map[string]string{
				"deleted": "false",
			},

			responseChatbots: []*chatbot.Chatbot{
				{
					ID: uuid.FromStringOrNil("31b00c64-f839-11ed-8f59-ab874a16ee9c"),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &chatbotHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatbotGets(ctx, tt.customerID, tt.size, tt.token, tt.filters).Return(tt.responseChatbots, nil)

			res, err := h.Gets(ctx, tt.customerID, tt.size, tt.token, tt.filters)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbots) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbots, res)
			}
		})
	}
}

func Test_Get(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseChatbot *chatbot.Chatbot
	}{
		{
			"normal",

			uuid.FromStringOrNil("a568e0b2-a70e-11ed-86c5-374896e473bd"),

			&chatbot.Chatbot{
				ID: uuid.FromStringOrNil("a568e0b2-a70e-11ed-86c5-374896e473bd"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &chatbotHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatbotGet(ctx, tt.id).Return(tt.responseChatbot, nil)

			res, err := h.Get(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbot) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbot, res)
			}
		})
	}
}

func Test_Delete(t *testing.T) {

	tests := []struct {
		name string

		id uuid.UUID

		responseChatbot *chatbot.Chatbot
	}{
		{
			"normal",

			uuid.FromStringOrNil("e7b895be-a710-11ed-9514-131c8c2fd995"),

			&chatbot.Chatbot{
				ID: uuid.FromStringOrNil("e7b895be-a710-11ed-9514-131c8c2fd995"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &chatbotHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatbotDelete(ctx, tt.id).Return(nil)
			mockDB.EXPECT().ChatbotGet(ctx, tt.id).Return(tt.responseChatbot, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbot.CustomerID, chatbot.EventTypeChatbotDeleted, tt.responseChatbot)

			res, err := h.Delete(ctx, tt.id)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbot) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbot, res)
			}
		})
	}
}

func Test_Update(t *testing.T) {

	tests := []struct {
		name string

		id          uuid.UUID
		chatbotName string
		detail      string
		engineType  chatbot.EngineType
		initPrompt  string

		responseChatbot *chatbot.Chatbot
	}{
		{
			name: "normal",

			id:          uuid.FromStringOrNil("fd49c1d6-f82e-11ed-8893-dfb489cd9bb9"),
			chatbotName: "new name",
			detail:      "new detail",
			engineType:  chatbot.EngineTypeChatGPT,
			initPrompt:  "new init prompt",

			responseChatbot: &chatbot.Chatbot{
				ID: uuid.FromStringOrNil("fd49c1d6-f82e-11ed-8893-dfb489cd9bb9"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockReq := requesthandler.NewMockRequestHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)

			h := &chatbotHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
			}

			ctx := context.Background()

			mockDB.EXPECT().ChatbotSetInfo(ctx, tt.id, tt.chatbotName, tt.detail, tt.engineType, tt.initPrompt).Return(nil)
			mockDB.EXPECT().ChatbotGet(ctx, tt.id).Return(tt.responseChatbot, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbot.CustomerID, chatbot.EventTypeChatbotUpdated, tt.responseChatbot)

			res, err := h.Update(ctx, tt.id, tt.chatbotName, tt.detail, tt.engineType, tt.initPrompt)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbot) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseChatbot, res)
			}
		})
	}
}

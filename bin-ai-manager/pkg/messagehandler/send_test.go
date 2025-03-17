package messagehandler

import (
	"context"
	"monorepo/bin-ai-manager/models/chatbot"
	"monorepo/bin-ai-manager/models/chatbotcall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/chatbotcallhandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/engine_dialogflow_handler"
	"monorepo/bin-ai-manager/pkg/engine_openai_handler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	"go.uber.org/mock/gomock"
)

func Test_sendChatGPT(t *testing.T) {

	tests := []struct {
		name string

		cc *chatbotcall.Chatbotcall

		responseMessages []*message.Message
		responseMessage  *message.Message

		expectSize     uint64
		expectFilters  map[string]string
		expectMessages []*message.Message
	}{
		{
			name: "normal",

			cc: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("595c0038-f2ba-11ef-8a26-4b552ba64340"),
				},
			},

			responseMessages: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("85133cbe-f2ba-11ef-9b51-6bf350630a68"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("85356ed8-f2ba-11ef-9bcb-63de90807209"),
					},
				},
			},
			responseMessage: &message.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8555f360-f2ba-11ef-ab46-fb44cb27875f"),
				},
			},

			expectSize: 1000,
			expectFilters: map[string]string{
				"deleted": "false",
			},
			expectMessages: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("85356ed8-f2ba-11ef-9bcb-63de90807209"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("85133cbe-f2ba-11ef-9b51-6bf350630a68"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockGPT := engine_openai_handler.NewMockEngineOpenaiHandler(mc)

			h := &messageHandler{
				utilHandler:   mockUtil,
				notifyHandler: mockNotify,
				db:            mockDB,

				engineOpenaiHandler: mockGPT,
			}

			ctx := context.Background()

			mockDB.EXPECT().MessageGets(ctx, tt.cc.ID, tt.expectSize, "", tt.expectFilters).Return(tt.responseMessages, nil)
			mockGPT.EXPECT().MessageSend(ctx, tt.cc, tt.expectMessages).Return(tt.responseMessage, nil)

			res, err := h.sendOpenai(ctx, tt.cc)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseMessage) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.responseMessage, res)
			}
		})
	}
}

func Test_Send_sendChatGPT(t *testing.T) {

	tests := []struct {
		name string

		chatbotcallID uuid.UUID
		role          message.Role
		content       string

		responseChatbotcall *chatbotcall.Chatbotcall
		responseUUID1       uuid.UUID
		responseUUID2       uuid.UUID

		responseMessages []*message.Message
		responseMessage  *message.Message

		expectMessage1 *message.Message
		expectMessage2 *message.Message

		expectSize     uint64
		expectFilters  map[string]string
		expectMessages []*message.Message
	}{
		{
			name: "normal",

			chatbotcallID: uuid.FromStringOrNil("76af2cf8-f2bc-11ef-bd4b-a7015b14c0f2"),
			role:          message.RoleUser,
			content:       "hello world!",

			responseChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("76af2cf8-f2bc-11ef-bd4b-a7015b14c0f2"),
					CustomerID: uuid.FromStringOrNil("7760703a-f2bc-11ef-b42a-33c238392350"),
				},
				Status:             chatbotcall.StatusProgressing,
				ChatbotEngineModel: chatbot.EngineModelOpenaiGPT3Dot5Turbo,
			},
			responseUUID1: uuid.FromStringOrNil("7734c35e-f2bc-11ef-a0ec-afc67dff1ffc"),
			responseUUID2: uuid.FromStringOrNil("7786dba8-f2bc-11ef-b9de-4b764cfeef4d"),

			responseMessages: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("85133cbe-f2ba-11ef-9b51-6bf350630a68"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("85356ed8-f2ba-11ef-9bcb-63de90807209"),
					},
				},
			},
			responseMessage: &message.Message{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8555f360-f2ba-11ef-ab46-fb44cb27875f"),
				},
				Role:    message.RoleAssistant,
				Content: "Hi there!",
			},

			expectMessage1: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("7734c35e-f2bc-11ef-a0ec-afc67dff1ffc"),
					CustomerID: uuid.FromStringOrNil("7760703a-f2bc-11ef-b42a-33c238392350"),
				},
				ChatbotcallID: uuid.FromStringOrNil("76af2cf8-f2bc-11ef-bd4b-a7015b14c0f2"),

				Direction: message.DirectionOutgoing,
				Role:      message.RoleUser,
				Content:   "hello world!",
			},
			expectMessage2: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("7786dba8-f2bc-11ef-b9de-4b764cfeef4d"),
					CustomerID: uuid.FromStringOrNil("7760703a-f2bc-11ef-b42a-33c238392350"),
				},
				ChatbotcallID: uuid.FromStringOrNil("76af2cf8-f2bc-11ef-bd4b-a7015b14c0f2"),

				Direction: message.DirectionIncoming,
				Role:      message.RoleAssistant,
				Content:   "Hi there!",
			},

			expectSize: 1000,
			expectFilters: map[string]string{
				"deleted": "false",
			},
			expectMessages: []*message.Message{
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("85356ed8-f2ba-11ef-9bcb-63de90807209"),
					},
				},
				{
					Identity: identity.Identity{
						ID: uuid.FromStringOrNil("85133cbe-f2ba-11ef-9b51-6bf350630a68"),
					},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChatbotcall := chatbotcallhandler.NewMockChatbotcallHandler(mc)
			mockGPT := engine_openai_handler.NewMockEngineOpenaiHandler(mc)

			h := &messageHandler{
				utilHandler:   mockUtil,
				notifyHandler: mockNotify,
				db:            mockDB,

				chatbotcallHandler:  mockChatbotcall,
				engineOpenaiHandler: mockGPT,
			}

			ctx := context.Background()

			mockChatbotcall.EXPECT().Get(ctx, tt.chatbotcallID).Return(tt.responseChatbotcall, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID1)
			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage1).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.responseUUID1).Return(tt.expectMessage1, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage1.CustomerID, message.EventTypeMessageCreated, tt.expectMessage1)

			mockDB.EXPECT().MessageGets(ctx, tt.responseChatbotcall.ID, tt.expectSize, "", tt.expectFilters).Return(tt.responseMessages, nil)
			mockGPT.EXPECT().MessageSend(ctx, tt.responseChatbotcall, tt.expectMessages).Return(tt.responseMessage, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID2)
			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage2).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.responseUUID2).Return(tt.expectMessage2, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage2.CustomerID, message.EventTypeMessageCreated, tt.expectMessage2)

			res, err := h.Send(ctx, tt.chatbotcallID, tt.role, tt.content)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectMessage1) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectMessage1, res)
			}
		})
	}
}

func Test_Send_sendDialogflow(t *testing.T) {

	tests := []struct {
		name string

		chatbotcallID uuid.UUID
		role          message.Role
		content       string

		responseChatbotcall *chatbotcall.Chatbotcall
		responseUUID1       uuid.UUID
		responseUUID2       uuid.UUID

		responseMessage1 *message.Message
		responseMessage2 *message.Message

		expectMessage1 *message.Message
		expectMessage2 *message.Message
	}{
		{
			name: "normal",

			chatbotcallID: uuid.FromStringOrNil("7dba479e-ff50-11ef-af5a-0b8ff2378435"),
			role:          message.RoleUser,
			content:       "hello world!",

			responseChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("7dba479e-ff50-11ef-af5a-0b8ff2378435"),
					CustomerID: uuid.FromStringOrNil("7e03ad6c-ff50-11ef-a910-efdcf54f7d9b"),
				},
				Status:             chatbotcall.StatusProgressing,
				ChatbotEngineModel: chatbot.EngineModelDialogflowES,
			},
			responseUUID1: uuid.FromStringOrNil("7e431876-ff50-11ef-a5ba-a7251571b293"),
			responseUUID2: uuid.FromStringOrNil("7e7594c2-ff50-11ef-93cf-9f3f35e9f012"),

			expectMessage1: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("7e431876-ff50-11ef-a5ba-a7251571b293"),
					CustomerID: uuid.FromStringOrNil("7e03ad6c-ff50-11ef-a910-efdcf54f7d9b"),
				},
				ChatbotcallID: uuid.FromStringOrNil("7dba479e-ff50-11ef-af5a-0b8ff2378435"),

				Direction: message.DirectionOutgoing,
				Role:      message.RoleUser,
				Content:   "hello world!",
			},
			expectMessage2: &message.Message{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("7e7594c2-ff50-11ef-93cf-9f3f35e9f012"),
					CustomerID: uuid.FromStringOrNil("7e03ad6c-ff50-11ef-a910-efdcf54f7d9b"),
				},
				ChatbotcallID: uuid.FromStringOrNil("7dba479e-ff50-11ef-af5a-0b8ff2378435"),

				Direction: message.DirectionIncoming,
				Role:      message.RoleAssistant,
				Content:   "Hi there!",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockUtil := utilhandler.NewMockUtilHandler(mc)
			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
			mockDB := dbhandler.NewMockDBHandler(mc)
			mockChatbotcall := chatbotcallhandler.NewMockChatbotcallHandler(mc)
			mockOpenai := engine_openai_handler.NewMockEngineOpenaiHandler(mc)
			mockDialogflow := engine_dialogflow_handler.NewMockEngineDialogflowHandler(mc)

			h := &messageHandler{
				utilHandler:   mockUtil,
				notifyHandler: mockNotify,
				db:            mockDB,

				chatbotcallHandler: mockChatbotcall,

				engineOpenaiHandler:     mockOpenai,
				engineDialogflowHandler: mockDialogflow,
			}

			ctx := context.Background()

			mockChatbotcall.EXPECT().Get(ctx, tt.chatbotcallID).Return(tt.responseChatbotcall, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID1)
			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage1).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.responseUUID1).Return(tt.expectMessage1, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage1.CustomerID, message.EventTypeMessageCreated, tt.expectMessage1)

			mockDialogflow.EXPECT().MessageSend(ctx, tt.responseChatbotcall, tt.expectMessage1).Return(tt.expectMessage2, nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUID2)
			mockDB.EXPECT().MessageCreate(ctx, tt.expectMessage2).Return(nil)
			mockDB.EXPECT().MessageGet(ctx, tt.responseUUID2).Return(tt.expectMessage2, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.expectMessage2.CustomerID, message.EventTypeMessageCreated, tt.expectMessage2)

			res, err := h.Send(ctx, tt.chatbotcallID, tt.role, tt.content)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectMessage1) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectMessage1, res)
			}
		})
	}
}

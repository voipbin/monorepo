package chatbotcallhandler

import (
	"context"
	"encoding/json"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/models/message"
	"monorepo/bin-chatbot-manager/pkg/chatbothandler"
	"monorepo/bin-chatbot-manager/pkg/dbhandler"
	"monorepo/bin-chatbot-manager/pkg/openai_handler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_ChatMessage(t *testing.T) {

	type testCase struct {
		name string

		chatbotcall *chatbotcall.Chatbotcall
		text        string

		responseMessage *message.Message

		expectRole message.Role
		expectText string
	}

	tests := []testCase{
		{
			name: "normal",
			chatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("02732972-96f1-4c51-9f76-38b32377493c"),
				},
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("0f7a3d29-fdb5-41ba-8fa9-3a85e02ce17a"),
				ChatbotEngineType: chatbot.EngineTypeNone,
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
				ReferenceID:       uuid.FromStringOrNil("some-reference-id"), // Add this
			},
			text: "hi",

			responseMessage: &message.Message{
				Role:    message.RoleAssistant, // Changed to assistant since the chatbot responds
				Content: "Hello, my name is chat-gpt.",
			},

			expectRole: message.RoleUser,
			expectText: "Hello, my name is chat-gpt.",
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
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)
			mockChatgpt := openai_handler.NewMockOpenaiHandler(mc)

			h := &chatbotcallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				chatbotHandler: mockChatbot,
				openaiHandler:  mockChatgpt,
			}
			ctx := context.Background()

			// Set up expectations for the mocks. Make sure arguments match what you're passing.
			mockReq.EXPECT().CallV1CallMediaStop(ctx, tt.chatbotcall.ReferenceID).Return(nil)
			mockReq.EXPECT().ChatbotV1MessageSend(ctx, tt.chatbotcall.ID, tt.expectRole, tt.text, gomock.Any()).Return(tt.responseMessage, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.chatbotcall.ReferenceID, tt.expectText, string(tt.chatbotcall.Gender), tt.chatbotcall.Language, 10000).Return(nil)

			if errChat := h.ChatMessage(ctx, tt.chatbotcall, tt.text); errChat != nil {
				t.Errorf("ChatMessage() error = %v", errChat)
			}
		})
	}
}

func Test_ChatInit(t *testing.T) {

	tests := []struct {
		name        string
		chatbot     *chatbot.Chatbot
		chatbotcall *chatbotcall.Chatbotcall
	}{
		{
			name: "normal",
			chatbot: &chatbot.Chatbot{
				EngineType: chatbot.EngineTypeNone,
				InitPrompt: "test message",
			},
			chatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("9bb7079c-f556-11ed-afbb-0f109793414b"),
					CustomerID: uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426614174000"),
				},
				ReferenceID: uuid.FromStringOrNil("55667788-9900-1122-3344-aabbccddeef1"),
				Gender:      chatbotcall.GenderNuetral,
				Language:    "en-US",
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
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)
			mockChatgpt := openai_handler.NewMockOpenaiHandler(mc)

			h := &chatbotcallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				chatbotHandler: mockChatbot,
				openaiHandler:  mockChatgpt,
			}
			ctx := context.Background()

			mockReq.EXPECT().ChatbotV1MessageSend(ctx, tt.chatbotcall.Identity.ID, message.RoleSystem, tt.chatbot.InitPrompt, gomock.Any()).Return(&message.Message{Content: "test assist"}, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.chatbotcall.ReferenceID, "test assist", string(tt.chatbotcall.Gender), tt.chatbotcall.Language, 10000).Return(nil)

			err := h.chatInit(ctx, tt.chatbot, tt.chatbotcall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_chatMessageActionsHandle(t *testing.T) {

	tests := []struct {
		name string

		chatbotcall *chatbotcall.Chatbotcall
		actions     []fmaction.Action
	}{
		{
			name: "normal",

			chatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("c243f296-fba3-11ed-b685-934f90d45843"),
				},
				ActiveflowID: uuid.FromStringOrNil("75496c7e-fba7-11ed-b6a8-f7993d25b0ab"),
			},
			actions: []fmaction.Action{
				{
					Type: fmaction.TypeAnswer,
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
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)
			mockChatgpt := openai_handler.NewMockOpenaiHandler(mc)

			h := &chatbotcallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				chatbotHandler: mockChatbot,
				openaiHandler:  mockChatgpt,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1ActiveflowPushActions(ctx, tt.chatbotcall.ActiveflowID, tt.actions).Return(&fmactiveflow.Activeflow{}, nil)
			mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, tt.chatbotcall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)

			if errHandle := h.chatMessageActionsHandle(ctx, tt.chatbotcall, tt.actions); errHandle != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errHandle)
			}
		})
	}
}

func Test_chatMessageReferenceTypeCall(t *testing.T) {

	tests := []struct {
		name           string
		chatbotcall    *chatbotcall.Chatbotcall
		messageContent string
	}{
		{
			name: "normal",
			chatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("47ea05dc-ef4c-11ef-8318-af1841553e05"),
					CustomerID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("48445be0-ef4c-11ef-9ac6-f39d9cadacfd"),
				ChatbotEngineType: chatbot.EngineTypeNone,
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
				ReferenceID:       uuid.FromStringOrNil("55667788-9900-1122-3344-aabbccddeef1"),
			},
			messageContent: "hi",
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
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)
			mockChatgpt := openai_handler.NewMockOpenaiHandler(mc)

			h := &chatbotcallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				chatbotHandler: mockChatbot,
				openaiHandler:  mockChatgpt,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallMediaStop(ctx, tt.chatbotcall.ReferenceID).Return(nil)
			mockReq.EXPECT().ChatbotV1MessageSend(ctx, tt.chatbotcall.Identity.ID, message.RoleUser, tt.messageContent, gomock.Any()).Return(&message.Message{Content: "test assist"}, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.chatbotcall.ReferenceID, "test assist", string(tt.chatbotcall.Gender), tt.chatbotcall.Language, 10000).Return(nil)

			errChat := h.chatMessageReferenceTypeCall(ctx, tt.chatbotcall, tt.messageContent)
			if errChat != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errChat)
			}
		})
	}
}

func Test_chatMessageHandle_actions(t *testing.T) {
	tests := []struct {
		name        string
		chatbotcall *chatbotcall.Chatbotcall
		message     *message.Message
	}{
		{
			name: "Action_Message",
			chatbotcall: &chatbotcall.Chatbotcall{
				ActiveflowID: uuid.FromStringOrNil("456789ab-1234-6543-3456-89abcdef0124"),
				ConfbridgeID: uuid.FromStringOrNil("11223344-5566-7788-9900-aabbccddeef2"),
			},
			message: &message.Message{
				Content: `[{"type": "some_action"}]`,
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
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)
			mockChatgpt := openai_handler.NewMockOpenaiHandler(mc)

			h := &chatbotcallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				chatbotHandler: mockChatbot,
				openaiHandler:  mockChatgpt,
			}
			ctx := context.Background()

			var tmpActions []fmaction.Action
			errUnmarshal := json.Unmarshal([]byte(tt.message.Content), &tmpActions)

			if errUnmarshal == nil {
				mockReq.EXPECT().FlowV1ActiveflowPushActions(ctx, tt.chatbotcall.ActiveflowID, gomock.Any()).Return(&fmactiveflow.Activeflow{}, nil)
				mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, tt.chatbotcall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)
			}

			errChat := h.chatMessageHandle(ctx, tt.chatbotcall, tt.message)
			if errChat != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errChat)
			}
		})
	}
}

func Test_chatMessageHandle_text(t *testing.T) {
	tests := []struct {
		name        string
		chatbotcall *chatbotcall.Chatbotcall
		message     *message.Message
	}{
		{
			name: "Text_Message",
			chatbotcall: &chatbotcall.Chatbotcall{
				ReferenceID: uuid.FromStringOrNil("55667788-9900-1122-3344-aabbccddeef1"),
				Gender:      chatbotcall.GenderNuetral,
				Language:    "ja-JP",
			},
			message: &message.Message{
				Content: "Hello",
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
			mockChatbot := chatbothandler.NewMockChatbotHandler(mc)
			mockChatgpt := openai_handler.NewMockOpenaiHandler(mc)

			h := &chatbotcallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				chatbotHandler: mockChatbot,
				openaiHandler:  mockChatgpt,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallTalk(ctx, tt.chatbotcall.ReferenceID, tt.message.Content, string(tt.chatbotcall.Gender), tt.chatbotcall.Language, 10000).Return(nil)

			errChat := h.chatMessageHandle(ctx, tt.chatbotcall, tt.message)
			if errChat != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errChat)
			}
		})
	}
}

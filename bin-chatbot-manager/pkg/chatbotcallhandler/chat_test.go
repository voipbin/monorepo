package chatbotcallhandler

import (
	"context"
	"testing"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"

	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/pkg/chatbothandler"
	"monorepo/bin-chatbot-manager/pkg/chatgpthandler"
	"monorepo/bin-chatbot-manager/pkg/dbhandler"
)

func Test_ChatMessage(t *testing.T) {

	tests := []struct {
		name string

		chatbotcall *chatbotcall.Chatbotcall
		message     string

		responseMessages    []chatbotcall.Message
		responseChatbotcall *chatbotcall.Chatbotcall

		expectText string
	}{
		{
			name: "normal",

			chatbotcall: &chatbotcall.Chatbotcall{
				ID:                uuid.FromStringOrNil("02732972-96f1-4c51-9f76-38b32377493c"),
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("0f7a3d29-fdb5-41ba-8fa9-3a85e02ce17a"),
				ChatbotEngineType: chatbot.EngineTypeChatGPT,
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
			},
			message: "Hi",

			responseMessages: []chatbotcall.Message{
				{
					Content: "hi",
				},
				{
					Content: "Hello, my name is chat-gpt.",
				},
			},
			responseChatbotcall: &chatbotcall.Chatbotcall{
				ID:                uuid.FromStringOrNil("02732972-96f1-4c51-9f76-38b32377493c"),
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("0f7a3d29-fdb5-41ba-8fa9-3a85e02ce17a"),
				ChatbotEngineType: chatbot.EngineTypeChatGPT,
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
				Messages: []chatbotcall.Message{
					{
						Content: "hi",
					},
					{
						Content: "Hello, my name is chat-gpt.",
					},
				},
			},

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
			mockChatgpt := chatgpthandler.NewMockChatgptHandler(mc)

			h := &chatbotcallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				chatbotHandler: mockChatbot,
				chatgptHandler: mockChatgpt,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallMediaStop(ctx, tt.chatbotcall.ReferenceID).Return(nil)
			mockChatgpt.EXPECT().ChatMessage(ctx, tt.chatbotcall.Messages, tt.message).Return(tt.responseMessages, nil)
			mockDB.EXPECT().ChatbotcallSetMessages(ctx, tt.chatbotcall.ID, tt.responseMessages)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.chatbotcall.ID).Return(tt.responseChatbotcall, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.chatbotcall.ReferenceID, tt.expectText, string(tt.chatbotcall.Gender), tt.chatbotcall.Language, 10000).Return(nil)

			if err := h.ChatMessage(ctx, tt.chatbotcall, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ChatInit(t *testing.T) {

	tests := []struct {
		name string

		chatbot     *chatbot.Chatbot
		chatbotcall *chatbotcall.Chatbotcall

		responseMessages []chatbotcall.Message
	}{
		{
			name: "normal",

			chatbot: &chatbot.Chatbot{
				EngineType: chatbot.EngineTypeChatGPT,
			},
			chatbotcall: &chatbotcall.Chatbotcall{
				ID: uuid.FromStringOrNil("9bb7079c-f556-11ed-afbb-0f109793414b"),
			},

			responseMessages: []chatbotcall.Message{
				{
					Role:    "system",
					Content: "test message",
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
			mockChatgpt := chatgpthandler.NewMockChatgptHandler(mc)

			h := &chatbotcallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				chatbotHandler: mockChatbot,
				chatgptHandler: mockChatgpt,
			}
			ctx := context.Background()

			mockChatgpt.EXPECT().ChatNew(ctx, tt.chatbot.InitPrompt).Return(tt.responseMessages, nil)
			mockDB.EXPECT().ChatbotcallSetMessages(ctx, tt.chatbotcall.ID, tt.responseMessages).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.chatbotcall.ID).Return(tt.chatbotcall, nil)

			if errInit := h.ChatInit(ctx, tt.chatbot, tt.chatbotcall); errInit != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errInit)
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
				ID:           uuid.FromStringOrNil("c243f296-fba3-11ed-b685-934f90d45843"),
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
			mockChatgpt := chatgpthandler.NewMockChatgptHandler(mc)

			h := &chatbotcallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				chatbotHandler: mockChatbot,
				chatgptHandler: mockChatgpt,
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

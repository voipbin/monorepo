package chatbotcallhandler

import (
	"context"
	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/pkg/chatbothandler"
	"monorepo/bin-chatbot-manager/pkg/chatgpthandler"
	"monorepo/bin-chatbot-manager/pkg/dbhandler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	"reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_messageSend(t *testing.T) {

	tests := []struct {
		name string

		chatbotcall *chatbotcall.Chatbotcall
		message     *chatbotcall.Message

		responseMessage     *chatbotcall.Message
		responseChatbotcall *chatbotcall.Chatbotcall

		expectMessages []chatbotcall.Message
		expectedRes    *chatbotcall.Chatbotcall
	}{
		{
			name: "normal",

			chatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8a238e4a-ef43-11ef-9c7a-53ede0c53f5c"),
				},
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("8a7a91f4-ef43-11ef-a68c-9fe04396763f"),
				ChatbotEngineType: chatbot.EngineTypeChatGPT,
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
			},
			message: &chatbotcall.Message{
				Role:    chatbotcall.MessageRoleUser,
				Content: "hi",
			},

			responseMessage: &chatbotcall.Message{
				Role:    chatbotcall.MessageRoleAssistant,
				Content: "Hello, my name is chat-gpt.",
			},
			responseChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8a238e4a-ef43-11ef-9c7a-53ede0c53f5c"),
				},
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("8a7a91f4-ef43-11ef-a68c-9fe04396763f"),
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

			expectMessages: []chatbotcall.Message{
				{
					Role:    chatbotcall.MessageRoleUser,
					Content: "hi",
				},
				{
					Role:    chatbotcall.MessageRoleAssistant,
					Content: "Hello, my name is chat-gpt.",
				},
			},
			expectedRes: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("8a238e4a-ef43-11ef-9c7a-53ede0c53f5c"),
				},
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("8a7a91f4-ef43-11ef-a68c-9fe04396763f"),
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

			mockChatgpt.EXPECT().ChatMessage(ctx, tt.chatbotcall, tt.message).Return(tt.responseMessage, nil)
			mockDB.EXPECT().ChatbotcallSetMessages(ctx, tt.chatbotcall.ID, tt.expectMessages)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.chatbotcall.ID).Return(tt.responseChatbotcall, nil)

			res, err := h.messageSend(ctx, tt.chatbotcall, tt.message)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectedRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectedRes, res)
			}
		})
	}
}

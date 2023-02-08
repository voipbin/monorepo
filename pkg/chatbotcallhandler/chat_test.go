package chatbotcallhandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"
	gomock "github.com/golang/mock/gomock"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/chatbothandler"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/chatgpthandler"
	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/pkg/dbhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/notifyhandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/requesthandler"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"
)

func Test_Chat(t *testing.T) {

	tests := []struct {
		name string

		chatbotcall *chatbotcall.Chatbotcall
		message     string

		responseChatbot *chatbot.Chatbot
		responseText    string
	}{
		{
			"normal",

			&chatbotcall.Chatbotcall{
				ID:            uuid.FromStringOrNil("02732972-96f1-4c51-9f76-38b32377493c"),
				ReferenceType: chatbotcall.ReferenceTypeCall,
				ChatbotID:     uuid.FromStringOrNil("0f7a3d29-fdb5-41ba-8fa9-3a85e02ce17a"),
				Gender:        chatbotcall.GenderFemale,
				Language:      "en-US",
			},
			"Hi",

			&chatbot.Chatbot{
				ID:         uuid.FromStringOrNil("0f7a3d29-fdb5-41ba-8fa9-3a85e02ce17a"),
				EngineType: chatbot.EngineTypeChatGPT,
			},
			"Hello, my name is chat-gpt.",
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
			mockChatbot.EXPECT().Get(ctx, tt.chatbotcall.ChatbotID).Return(tt.responseChatbot, nil)
			mockChatgpt.EXPECT().Chat(ctx, tt.message).Return(tt.responseText, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.chatbotcall.ReferenceID, tt.responseText, string(tt.chatbotcall.Gender), tt.chatbotcall.Language, 10000).Return(nil)

			if err := h.Chat(ctx, tt.chatbotcall, tt.message); err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

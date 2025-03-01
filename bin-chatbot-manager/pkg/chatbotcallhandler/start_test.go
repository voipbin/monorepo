package chatbotcallhandler

import (
	"context"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
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
	reflect "reflect"
	"testing"
	"time"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_startReferenceTypeCall(t *testing.T) {
	tests := []struct {
		name string

		chatbot      *chatbot.Chatbot
		activeflowID uuid.UUID
		referenceID  uuid.UUID
		gender       chatbotcall.Gender
		language     string

		responseConfbridge      *cmconfbridge.Confbridge
		responseUUIDChatbotcall uuid.UUID
		responseChatbotcall     *chatbotcall.Chatbotcall
		responseMessage         *message.Message

		expectChatbotcall         *chatbotcall.Chatbotcall
		expectChatbotcallMessages []message.Message
		expectMessage             *message.Message
		expectRes                 *chatbotcall.Chatbotcall
	}{
		{
			name: "normal",
			chatbot: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				EngineType: chatbot.EngineTypeNone,
				InitPrompt: "hello, this is init prompt message.",
			},
			activeflowID: uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
			referenceID:  uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
			gender:       chatbotcall.GenderFemale,
			language:     "en-US",

			responseConfbridge: &cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
			},
			responseUUIDChatbotcall: uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
			responseChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
				},
				ReferenceType: chatbotcall.ReferenceTypeCall,
				ConfbridgeID:  uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
			},
			responseMessage: &message.Message{
				Role:    "assistant",
				Content: "test assistant message.",
			},
			expectChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				ChatbotID:         uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
				ActiveflowID:      uuid.FromStringOrNil("a47265a2-f06d-11ef-8317-2bf92ae88a9d"),
				ChatbotEngineType: chatbot.EngineTypeNone,
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
				ConfbridgeID:      uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
				Status:            chatbotcall.StatusInitiating,
			},
			expectMessage: &message.Message{
				Role:    message.RoleSystem,
				Content: "hello, this is init prompt message.",
			},
			expectChatbotcallMessages: []message.Message{
				{Role: "system", Content: "hello, this is init prompt message."},
				{Role: "assistant", Content: "test assistant message."},
			},
			expectRes: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
				},
				ReferenceType: chatbotcall.ReferenceTypeCall,
				ConfbridgeID:  uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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
			mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, tt.chatbot.CustomerID, cmconfbridge.TypeConference).Return(tt.responseConfbridge, nil)

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChatbotcall)
			mockDB.EXPECT().ChatbotcallCreate(ctx, tt.expectChatbotcall).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseUUIDChatbotcall).Return(tt.responseChatbotcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallInitializing, tt.responseChatbotcall)

			mockReq.EXPECT().ChatbotV1MessageSend(ctx, tt.responseChatbotcall.ID, message.RoleSystem, tt.chatbot.InitPrompt, 30000).Return(tt.responseMessage, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.responseChatbotcall.ReferenceID, tt.responseMessage.Content, string(tt.responseChatbotcall.Gender), tt.responseChatbotcall.Language, 10000).Return(nil)

			res, err := h.startReferenceTypeCall(ctx, tt.chatbot, tt.activeflowID, tt.referenceID, tt.gender, tt.language)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			time.Sleep(100 * time.Millisecond)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_startReferenceTypeNone(t *testing.T) {
	tests := []struct {
		name string

		chatbot  *chatbot.Chatbot
		gender   chatbotcall.Gender
		language string

		responseUUIDChatbotcall uuid.UUID
		responseChatbotcall     *chatbotcall.Chatbotcall
		responseMessage         *message.Message

		expectChatbotcall *chatbotcall.Chatbotcall
		expectRes         *chatbotcall.Chatbotcall
	}{
		{
			name: "normal",
			chatbot: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("1d758ff0-f06f-11ef-bcb1-1ff1f3691915"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				EngineType: chatbot.EngineTypeNone,
				InitPrompt: "hello, this is init prompt message.",
			},
			gender:   chatbotcall.GenderFemale,
			language: "en-US",

			responseUUIDChatbotcall: uuid.FromStringOrNil("1e1a95ea-f06f-11ef-b98e-cf0423a1e383"),
			responseChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("1e1a95ea-f06f-11ef-b98e-cf0423a1e383"),
				},
			},
			responseMessage: &message.Message{
				Role:    "assistant",
				Content: "test assistant message.",
			},
			expectChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("1e1a95ea-f06f-11ef-b98e-cf0423a1e383"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				ChatbotID:         uuid.FromStringOrNil("1d758ff0-f06f-11ef-bcb1-1ff1f3691915"),
				ChatbotEngineType: chatbot.EngineTypeNone,
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
				Status:            chatbotcall.StatusInitiating,
			},
			expectRes: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("1e1a95ea-f06f-11ef-b98e-cf0423a1e383"),
				},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
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

			// create
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChatbotcall)
			mockDB.EXPECT().ChatbotcallCreate(ctx, tt.expectChatbotcall).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseUUIDChatbotcall).Return(tt.responseChatbotcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallInitializing, tt.responseChatbotcall)

			mockReq.EXPECT().ChatbotV1MessageSend(ctx, tt.responseChatbotcall.ID, message.RoleSystem, tt.chatbot.InitPrompt, 30000).Return(tt.responseMessage, nil)

			mockDB.EXPECT().ChatbotcallUpdateStatusProgressing(ctx, tt.responseChatbotcall.ID, uuid.Nil).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseChatbotcall.ID).Return(tt.responseChatbotcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallProgressing, tt.responseChatbotcall)

			res, err := h.startReferenceTypeNone(ctx, tt.chatbot, tt.gender, tt.language)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
			time.Sleep(100 * time.Millisecond)
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("expected: %v, got: %v", tt.expectRes, res)
			}
		})
	}
}

package chatbotcallhandler

import (
	"context"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
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
		responseMessage         *chatbotcall.Message

		expectChatbotcall         *chatbotcall.Chatbotcall
		expectChatbotcallMessages []chatbotcall.Message
		expectMessage             *chatbotcall.Message
		expectRes                 *chatbotcall.Chatbotcall
	}{
		{
			name: "normal",

			chatbot: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("a4107e6e-f06d-11ef-9b7a-03c848b3bb41"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				EngineType: chatbot.EngineTypeChatGPT,
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
				ConfbridgeID: uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
			},
			responseMessage: &chatbotcall.Message{
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
				ChatbotEngineType: chatbot.EngineTypeChatGPT,
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("a4a663fc-f06d-11ef-aeb9-6b2d8f0da3ac"),
				ConfbridgeID:      uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
				Messages:          []chatbotcall.Message{},
				Status:            chatbotcall.StatusInitiating,
			},
			expectMessage: &chatbotcall.Message{
				Role:    chatbotcall.MessageRoleSystem,
				Content: "hello, this is init prompt message.",
			},
			expectChatbotcallMessages: []chatbotcall.Message{
				{
					Role:    "system",
					Content: "hello, this is init prompt message.",
				},
				{
					Role:    "assistant",
					Content: "test assistant message.",
				},
			},
			expectRes: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
				},
				ConfbridgeID: uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
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

			mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, tt.chatbot.CustomerID, cmconfbridge.TypeConference).Return(tt.responseConfbridge, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChatbotcall)
			mockDB.EXPECT().ChatbotcallCreate(ctx, tt.expectChatbotcall).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseUUIDChatbotcall).Return(tt.responseChatbotcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallInitializing, tt.responseChatbotcall)

			mockChatgpt.EXPECT().ChatNew(ctx, tt.responseChatbotcall, tt.expectMessage).Return(tt.responseMessage, nil)
			mockDB.EXPECT().ChatbotcallSetMessages(ctx, tt.responseChatbotcall.ID, tt.expectChatbotcallMessages).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseUUIDChatbotcall).Return(tt.responseChatbotcall, nil)

			res, err := h.startReferenceTypeCall(ctx, tt.chatbot, tt.activeflowID, tt.referenceID, tt.gender, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

func Test_startReferenceTypeNone(t *testing.T) {

	tests := []struct {
		name string

		chatbot      *chatbot.Chatbot
		activeflowID uuid.UUID
		referenceID  uuid.UUID
		gender       chatbotcall.Gender
		language     string

		responseUUIDChatbotcall uuid.UUID
		responseChatbotcall     *chatbotcall.Chatbotcall
		responseMessage         *chatbotcall.Message

		expectChatbotcall         *chatbotcall.Chatbotcall
		expectChatbotcallMessages []chatbotcall.Message
		expectMessage             *chatbotcall.Message
		expectRes                 *chatbotcall.Chatbotcall
	}{
		{
			name: "normal",

			chatbot: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("1d758ff0-f06f-11ef-bcb1-1ff1f3691915"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				EngineType: chatbot.EngineTypeChatGPT,
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
			responseMessage: &chatbotcall.Message{
				Role:    "assistant",
				Content: "test assistant message.",
			},

			expectChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("1e1a95ea-f06f-11ef-b98e-cf0423a1e383"),
					CustomerID: uuid.FromStringOrNil("1dbecf3a-f06f-11ef-bb0a-bfec64e31a47"),
				},
				ChatbotID:         uuid.FromStringOrNil("1d758ff0-f06f-11ef-bcb1-1ff1f3691915"),
				ChatbotEngineType: chatbot.EngineTypeChatGPT,
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
				Messages:          []chatbotcall.Message{},
				Status:            chatbotcall.StatusInitiating,
			},
			expectMessage: &chatbotcall.Message{
				Role:    chatbotcall.MessageRoleSystem,
				Content: "hello, this is init prompt message.",
			},
			expectChatbotcallMessages: []chatbotcall.Message{
				{
					Role:    "system",
					Content: "hello, this is init prompt message.",
				},
				{
					Role:    "assistant",
					Content: "test assistant message.",
				},
			},
			expectRes: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("1e1a95ea-f06f-11ef-b98e-cf0423a1e383"),
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

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChatbotcall)
			mockDB.EXPECT().ChatbotcallCreate(ctx, tt.expectChatbotcall).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseUUIDChatbotcall).Return(tt.responseChatbotcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallInitializing, tt.responseChatbotcall)

			mockChatgpt.EXPECT().ChatNew(ctx, tt.responseChatbotcall, tt.expectMessage).Return(tt.responseMessage, nil)
			mockDB.EXPECT().ChatbotcallSetMessages(ctx, tt.responseChatbotcall.ID, tt.expectChatbotcallMessages).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseUUIDChatbotcall).Return(tt.responseChatbotcall, nil)

			res, err := h.startReferenceTypeNone(ctx, tt.chatbot, tt.gender, tt.language)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			time.Sleep(time.Millisecond * 100)

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match.\nexpect: %v\ngot: %v", tt.expectRes, res)
			}
		})
	}
}

package chatbotcallhandler

import (
	"context"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/models/message"
	"monorepo/bin-chatbot-manager/models/service"
	"monorepo/bin-chatbot-manager/pkg/chatbothandler"
	"monorepo/bin-chatbot-manager/pkg/dbhandler"
	"monorepo/bin-chatbot-manager/pkg/openai_handler"
	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	fmaction "monorepo/bin-flow-manager/models/action"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_ServiceStart(t *testing.T) {
	tests := []struct {
		name string

		chatbotID     uuid.UUID
		activeflowID  uuid.UUID
		referenceType chatbotcall.ReferenceType
		referenceID   uuid.UUID
		gender        chatbotcall.Gender
		language      string

		responseChatbot         *chatbot.Chatbot
		responseConfbridge      *cmconfbridge.Confbridge
		responseUUIDChatbotcall uuid.UUID
		responseChatbotcall     *chatbotcall.Chatbotcall
		responseMessage         *message.Message
		responseUUIDAction      uuid.UUID

		expectChatbotcall *chatbotcall.Chatbotcall
		expectRes         *service.Service
	}{
		{
			name:          "normal - english female",
			chatbotID:     uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
			activeflowID:  uuid.FromStringOrNil("45357f3e-fba5-11ed-aec8-f3762a730824"),
			referenceType: chatbotcall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("3b86f912-a459-4fd8-80ec-e6b632a2150a"),
			gender:        chatbotcall.GenderFemale,
			language:      "en-US",

			responseChatbot: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				EngineType: chatbot.EngineTypeNone,
				InitPrompt: "hello, this is init prompt message.",
			},
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
			responseMessage: &message.Message{
				Role:    "assistant",
				Content: "test assistant message.",
			},
			responseUUIDAction: uuid.FromStringOrNil("5001add9-0806-4adf-a535-15fc220a2019"),

			expectChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
					CustomerID: uuid.FromStringOrNil("483054da-13f5-42de-a785-dc20598726c1"),
				},
				ChatbotID:         uuid.FromStringOrNil("90560847-44bf-44ee-a28e-b7e86a488450"),
				ActiveflowID:      uuid.FromStringOrNil("45357f3e-fba5-11ed-aec8-f3762a730824"),
				ChatbotEngineType: chatbot.EngineTypeNone,
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("3b86f912-a459-4fd8-80ec-e6b632a2150a"),
				ConfbridgeID:      uuid.FromStringOrNil("ec6d153d-dd5a-4eef-bc27-8fcebe100704"),
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
				Messages:          []chatbotcall.Message{},
				Status:            chatbotcall.StatusInitiating,
			},
			expectRes: &service.Service{
				ID:   uuid.FromStringOrNil("a6cd01d0-d785-467f-9069-684e46cc2644"),
				Type: service.TypeChatbotcall,
				PushActions: []fmaction.Action{
					{
						ID:     uuid.FromStringOrNil("5001add9-0806-4adf-a535-15fc220a2019"),
						Type:   fmaction.TypeConfbridgeJoin,
						Option: []byte(`{"confbridge_id":"ec6d153d-dd5a-4eef-bc27-8fcebe100704"}`),
					},
				},
			},
		},
		{
			name:          "normal - french male",
			chatbotID:     uuid.FromStringOrNil("d1a7a4f8-7c9f-4b8a-9e60-3f7f6b6e09de"),
			activeflowID:  uuid.FromStringOrNil("b8a9c41a-9a8f-4d7f-a4fa-c123456789ab"),
			referenceType: chatbotcall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("e3b8a4f8-7c9f-4b8a-9e60-3f7f6b6e09de"),
			gender:        chatbotcall.GenderMale,
			language:      "fr-FR",

			responseChatbot: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("d1a7a4f8-7c9f-4b8a-9e60-3f7f6b6e09de"),
					CustomerID: uuid.FromStringOrNil("f1c7a4f8-7c9f-4b8a-9e60-3f7f6b6e09de"),
				},
				EngineType: chatbot.EngineTypeNone,
				InitPrompt: "bonjour, ceci est le message d'initialisation.",
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("a2b3c4d5-e6f7-4a8b-9c0d-1e2f3a4b5c6d"),
			},
			responseUUIDChatbotcall: uuid.FromStringOrNil("c3d4e5f6-7a8b-4c9d-0e1f-2a3b4c5d6e7f"),
			responseChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("c3d4e5f6-7a8b-4c9d-0e1f-2a3b4c5d6e7f"),
				},
				ConfbridgeID: uuid.FromStringOrNil("a2b3c4d5-e6f7-4a8b-9c0d-1e2f3a4b5c6d"),
			},
			responseMessage: &message.Message{
				Role:    "assistant",
				Content: "message d'assistant de test.",
			},
			responseUUIDAction: uuid.FromStringOrNil("d4e5f6a7-8b9c-4d0e-1f2a-3b4c5d6e7f8a"),

			expectChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("c3d4e5f6-7a8b-4c9d-0e1f-2a3b4c5d6e7f"),
					CustomerID: uuid.FromStringOrNil("f1c7a4f8-7c9f-4b8a-9e60-3f7f6b6e09de"),
				},
				ChatbotID:         uuid.FromStringOrNil("d1a7a4f8-7c9f-4b8a-9e60-3f7f6b6e09de"),
				ActiveflowID:      uuid.FromStringOrNil("b8a9c41a-9a8f-4d7f-a4fa-c123456789ab"),
				ChatbotEngineType: chatbot.EngineTypeNone,
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("e3b8a4f8-7c9f-4b8a-9e60-3f7f6b6e09de"),
				ConfbridgeID:      uuid.FromStringOrNil("a2b3c4d5-e6f7-4a8b-9c0d-1e2f3a4b5c6d"),
				Gender:            chatbotcall.GenderMale,
				Language:          "fr-FR",
				Messages:          []chatbotcall.Message{},
				Status:            chatbotcall.StatusInitiating,
			},
			expectRes: &service.Service{
				ID:   uuid.FromStringOrNil("c3d4e5f6-7a8b-4c9d-0e1f-2a3b4c5d6e7f"),
				Type: service.TypeChatbotcall,
				PushActions: []fmaction.Action{
					{
						ID:     uuid.FromStringOrNil("d4e5f6a7-8b9c-4d0e-1f2a-3b4c5d6e7f8a"),
						Type:   fmaction.TypeConfbridgeJoin,
						Option: []byte(`{"confbridge_id":"a2b3c4d5-e6f7-4a8b-9c0d-1e2f3a4b5c6d"}`),
					},
				},
			},
		},
		{
			name:          "normal - english male",
			chatbotID:     uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555"),
			activeflowID:  uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa"),
			referenceType: chatbotcall.ReferenceTypeCall,
			referenceID:   uuid.FromStringOrNil("bbbbbbbb-cccc-dddd-eeee-ffffffffffff"),
			gender:        chatbotcall.GenderMale,
			language:      "en-US",

			responseChatbot: &chatbot.Chatbot{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555"),
					CustomerID: uuid.FromStringOrNil("99999999-8888-7777-6666-555555555555"),
				},
				EngineType: chatbot.EngineTypeNone,
				InitPrompt: "hello, this is another init prompt message.",
			},
			responseConfbridge: &cmconfbridge.Confbridge{
				ID: uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
			},
			responseUUIDChatbotcall: uuid.FromStringOrNil("cccccccc-dddd-eeee-ffff-000000000000"),
			responseChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("cccccccc-dddd-eeee-ffff-000000000000"),
				},
				ConfbridgeID: uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
			},
			responseMessage: &message.Message{
				Role:    "assistant",
				Content: "another test assistant message.",
			},
			responseUUIDAction: uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-111111111111"),

			expectChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("cccccccc-dddd-eeee-ffff-000000000000"),
					CustomerID: uuid.FromStringOrNil("99999999-8888-7777-6666-555555555555"),
				},
				ChatbotID:         uuid.FromStringOrNil("11111111-2222-3333-4444-555555555555"),
				ActiveflowID:      uuid.FromStringOrNil("66666666-7777-8888-9999-aaaaaaaaaaaa"),
				ChatbotEngineType: chatbot.EngineTypeNone,
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("bbbbbbbb-cccc-dddd-eeee-ffffffffffff"),
				ConfbridgeID:      uuid.FromStringOrNil("aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"),
				Gender:            chatbotcall.GenderMale,
				Language:          "en-US",
				Messages:          []chatbotcall.Message{},
				Status:            chatbotcall.StatusInitiating,
			},
			expectRes: &service.Service{
				ID:   uuid.FromStringOrNil("cccccccc-dddd-eeee-ffff-000000000000"),
				Type: service.TypeChatbotcall,
				PushActions: []fmaction.Action{
					{
						ID:     uuid.FromStringOrNil("dddddddd-eeee-ffff-0000-111111111111"),
						Type:   fmaction.TypeConfbridgeJoin,
						Option: []byte(`{"confbridge_id":"aaaaaaaa-bbbb-cccc-dddd-eeeeeeeeeeee"}`),
					},
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
			mockChatbot.EXPECT().Get(ctx, tt.chatbotID).Return(tt.responseChatbot, nil)
			mockReq.EXPECT().CallV1ConfbridgeCreate(ctx, tt.responseChatbot.CustomerID, cmconfbridge.TypeConference).Return(tt.responseConfbridge, nil)
			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDChatbotcall)
			mockDB.EXPECT().ChatbotcallCreate(ctx, tt.expectChatbotcall).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseUUIDChatbotcall).Return(tt.responseChatbotcall, nil)
			mockNotify.EXPECT().PublishWebhookEvent(ctx, tt.responseChatbotcall.CustomerID, chatbotcall.EventTypeChatbotcallInitializing, tt.responseChatbotcall)

			mockReq.EXPECT().ChatbotV1MessageSend(ctx, tt.responseChatbotcall.ID, message.RoleSystem, tt.responseChatbot.InitPrompt, 30000).Return(tt.responseMessage, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.responseChatbotcall.ReferenceID, tt.responseMessage.Content, string(tt.responseChatbotcall.Gender), tt.responseChatbotcall.Language, 10000).Return(nil)

			mockUtil.EXPECT().UUIDCreate().Return(tt.responseUUIDAction)

			res, err := h.ServiceStart(ctx, tt.chatbotID, tt.activeflowID, tt.referenceType, tt.referenceID, tt.gender, tt.language)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Expected result %#v, got %#v", tt.expectRes, res)
			}
		})
	}
}

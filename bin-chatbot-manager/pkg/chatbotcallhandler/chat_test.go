package chatbotcallhandler

import (
	"context"
	"reflect"
	"testing"

	cmconfbridge "monorepo/bin-call-manager/models/confbridge"

	"monorepo/bin-common-handler/models/identity"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"

	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-chatbot-manager/models/chatbotcall"
	"monorepo/bin-chatbot-manager/pkg/chatbothandler"
	"monorepo/bin-chatbot-manager/pkg/chatgpthandler"
	"monorepo/bin-chatbot-manager/pkg/dbhandler"
)

func Test_ChatMessageByID(t *testing.T) {

	tests := []struct {
		name string

		id   uuid.UUID
		role chatbotcall.MessageRole
		text string

		responseChatbotcall_1 *chatbotcall.Chatbotcall
		responseChatbotcall_2 *chatbotcall.Chatbotcall
		responseMessage       *chatbotcall.Message

		expectMessage  *chatbotcall.Message
		expectMessages []chatbotcall.Message
		expectText     string
	}{
		{
			name: "normal",

			id:   uuid.FromStringOrNil("12038692-efa0-11ef-a819-6bffa7999473"),
			role: chatbotcall.MessageRoleUser,
			text: "hi",

			responseChatbotcall_1: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("12038692-efa0-11ef-a819-6bffa7999473"),
				},
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("0f7a3d29-fdb5-41ba-8fa9-3a85e02ce17a"),
				ChatbotEngineType: chatbot.EngineTypeChatGPT,
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
				Messages: []chatbotcall.Message{
					{
						Role:    chatbotcall.MessageRoleSystem,
						Content: "test system message",
					},
				},
			},
			responseChatbotcall_2: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("12038692-efa0-11ef-a819-6bffa7999473"),
				},
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("0f7a3d29-fdb5-41ba-8fa9-3a85e02ce17a"),
				ChatbotEngineType: chatbot.EngineTypeChatGPT,
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
				Messages: []chatbotcall.Message{
					{
						Role:    chatbotcall.MessageRoleSystem,
						Content: "test system message",
					},
					{
						Role:    chatbotcall.MessageRoleUser,
						Content: "hi",
					},
					{
						Role:    chatbotcall.MessageRoleAssistant,
						Content: "Hello, my name is chat-gpt.",
					},
				},
			},
			responseMessage: &chatbotcall.Message{
				Role:    chatbotcall.MessageRoleAssistant,
				Content: "Hello, my name is chat-gpt.",
			},

			expectMessage: &chatbotcall.Message{
				Role:    chatbotcall.MessageRoleUser,
				Content: "hi",
			},
			expectMessages: []chatbotcall.Message{
				{
					Role:    chatbotcall.MessageRoleSystem,
					Content: "test system message",
				},
				{
					Role:    chatbotcall.MessageRoleUser,
					Content: "hi",
				},
				{
					Role:    chatbotcall.MessageRoleAssistant,
					Content: "Hello, my name is chat-gpt.",
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

			mockDB.EXPECT().ChatbotcallGet(ctx, tt.id).Return(tt.responseChatbotcall_1, nil)
			mockReq.EXPECT().CallV1CallMediaStop(ctx, tt.responseChatbotcall_1.ReferenceID).Return(nil)
			mockChatgpt.EXPECT().ChatMessage(ctx, tt.responseChatbotcall_1, tt.expectMessage).Return(tt.responseMessage, nil)
			mockDB.EXPECT().ChatbotcallSetMessages(ctx, tt.responseChatbotcall_2.ID, tt.expectMessages)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.responseChatbotcall_2.ID).Return(tt.responseChatbotcall_2, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.responseChatbotcall_2.ReferenceID, tt.expectText, string(tt.responseChatbotcall_2.Gender), tt.responseChatbotcall_2.Language, 10000).Return(nil)

			res, err := h.ChatMessageByID(ctx, tt.id, tt.role, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbotcall_2) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseChatbotcall_2, res)
			}
		})
	}
}

func Test_ChatMessage(t *testing.T) {

	tests := []struct {
		name string

		chatbotcall *chatbotcall.Chatbotcall
		role        chatbotcall.MessageRole
		text        string

		responseChatbotcall *chatbotcall.Chatbotcall
		responseMessage     *chatbotcall.Message

		expectMessage  *chatbotcall.Message
		expectMessages []chatbotcall.Message
		expectText     string
	}{
		{
			name: "normal",

			chatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("02732972-96f1-4c51-9f76-38b32377493c"),
				},
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("0f7a3d29-fdb5-41ba-8fa9-3a85e02ce17a"),
				ChatbotEngineType: chatbot.EngineTypeChatGPT,
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
			},
			role: chatbotcall.MessageRoleUser,
			text: "hi",

			responseChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("02732972-96f1-4c51-9f76-38b32377493c"),
				},
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
			responseMessage: &chatbotcall.Message{
				Role:    chatbotcall.MessageRoleAssistant,
				Content: "Hello, my name is chat-gpt.",
			},

			expectMessage: &chatbotcall.Message{
				Role:    chatbotcall.MessageRoleUser,
				Content: "hi",
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
			mockChatgpt.EXPECT().ChatMessage(ctx, tt.chatbotcall, tt.expectMessage).Return(tt.responseMessage, nil)
			mockDB.EXPECT().ChatbotcallSetMessages(ctx, tt.chatbotcall.ID, tt.expectMessages)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.chatbotcall.ID).Return(tt.responseChatbotcall, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.chatbotcall.ReferenceID, tt.expectText, string(tt.chatbotcall.Gender), tt.chatbotcall.Language, 10000).Return(nil)

			res, err := h.ChatMessage(ctx, tt.chatbotcall, tt.role, tt.text)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbotcall) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseChatbotcall, res)
			}
		})
	}
}

func Test_ChatInit(t *testing.T) {

	tests := []struct {
		name string

		chatbot     *chatbot.Chatbot
		chatbotcall *chatbotcall.Chatbotcall

		responseMessage *chatbotcall.Message

		expectMessage  *chatbotcall.Message
		expectMessages []chatbotcall.Message
		expectRes      *chatbotcall.Chatbotcall
	}{
		{
			name: "normal",

			chatbot: &chatbot.Chatbot{
				EngineType: chatbot.EngineTypeChatGPT,
				InitPrompt: "test message",
			},
			chatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("9bb7079c-f556-11ed-afbb-0f109793414b"),
				},
			},

			responseMessage: &chatbotcall.Message{
				Role:    chatbotcall.MessageRoleAssistant,
				Content: "test assist",
			},

			expectMessage: &chatbotcall.Message{
				Role:    chatbotcall.MessageRoleSystem,
				Content: "test message",
			},
			expectMessages: []chatbotcall.Message{
				{
					Role:    chatbotcall.MessageRoleSystem,
					Content: "test message",
				},
				{
					Role:    chatbotcall.MessageRoleAssistant,
					Content: "test assist",
				},
			},
			expectRes: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("9bb7079c-f556-11ed-afbb-0f109793414b"),
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

			mockChatgpt.EXPECT().ChatNew(ctx, tt.chatbotcall, tt.expectMessage).Return(tt.responseMessage, nil)
			mockDB.EXPECT().ChatbotcallSetMessages(ctx, tt.chatbotcall.ID, tt.expectMessages).Return(nil)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.chatbotcall.ID).Return(tt.chatbotcall, nil)

			res, err := h.chatInit(ctx, tt.chatbot, tt.chatbotcall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.expectRes) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectRes, res)
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

func Test_chatMessageReferenceTypeCall(t *testing.T) {

	tests := []struct {
		name string

		chatbotcall *chatbotcall.Chatbotcall
		message     *chatbotcall.Message

		responseChatbotcall *chatbotcall.Chatbotcall
		responseMessage     *chatbotcall.Message

		expectMessages []chatbotcall.Message
		expectText     string
	}{
		{
			name: "normal",

			chatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("47ea05dc-ef4c-11ef-8318-af1841553e05"),
				},
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("48445be0-ef4c-11ef-9ac6-f39d9cadacfd"),
				ChatbotEngineType: chatbot.EngineTypeChatGPT,
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
			},
			message: &chatbotcall.Message{
				Role:    chatbotcall.MessageRoleUser,
				Content: "hi",
			},

			responseChatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("47ea05dc-ef4c-11ef-8318-af1841553e05"),
				},
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("48445be0-ef4c-11ef-9ac6-f39d9cadacfd"),
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
			responseMessage: &chatbotcall.Message{
				Content: "Hello, my name is chat-gpt.",
			},

			expectMessages: []chatbotcall.Message{
				{
					Role:    chatbotcall.MessageRoleUser,
					Content: "hi",
				},
				{
					Content: "Hello, my name is chat-gpt.",
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
			mockChatgpt.EXPECT().ChatMessage(ctx, tt.chatbotcall, tt.message).Return(tt.responseMessage, nil)
			mockDB.EXPECT().ChatbotcallSetMessages(ctx, tt.chatbotcall.ID, tt.expectMessages)
			mockDB.EXPECT().ChatbotcallGet(ctx, tt.chatbotcall.ID).Return(tt.responseChatbotcall, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.chatbotcall.ReferenceID, tt.expectText, string(tt.chatbotcall.Gender), tt.chatbotcall.Language, 10000).Return(nil)

			res, err := h.chatMessageReferenceTypeCall(ctx, tt.chatbotcall, tt.message)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbotcall) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseChatbotcall, res)
			}
		})
	}
}

func Test_chatMessageReferenceTypeNone(t *testing.T) {

	tests := []struct {
		name string

		chatbotcall *chatbotcall.Chatbotcall
		message     *chatbotcall.Message

		responseMessage     *chatbotcall.Message
		responseChatbotcall *chatbotcall.Chatbotcall

		expectMessages []chatbotcall.Message
	}{
		{
			name: "normal",

			chatbotcall: &chatbotcall.Chatbotcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("ac59d614-ef4c-11ef-92c4-d3fac6f89f7c"),
				},
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("ac802da0-ef4c-11ef-ae2e-4bb145b60231"),
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
					ID: uuid.FromStringOrNil("ac59d614-ef4c-11ef-92c4-d3fac6f89f7c"),
				},
				ReferenceType:     chatbotcall.ReferenceTypeCall,
				ChatbotID:         uuid.FromStringOrNil("ac802da0-ef4c-11ef-ae2e-4bb145b60231"),
				ChatbotEngineType: chatbot.EngineTypeChatGPT,
				Gender:            chatbotcall.GenderFemale,
				Language:          "en-US",
				Messages: []chatbotcall.Message{
					{
						Role:    chatbotcall.MessageRoleUser,
						Content: "hi",
					},
					{
						Role:    chatbotcall.MessageRoleAssistant,
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

			res, err := h.chatMessageReferenceTypeNone(ctx, tt.chatbotcall, tt.message)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}

			if !reflect.DeepEqual(res, tt.responseChatbotcall) {
				t.Errorf("Wrong match. expect: %v, got: %v", tt.responseChatbotcall, res)
			}
		})
	}
}

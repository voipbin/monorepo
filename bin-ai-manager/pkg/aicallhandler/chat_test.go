package aicallhandler

import (
	"context"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
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

		aicall *aicall.AIcall
		text   string

		responseMessage *message.Message

		expectRole message.Role
		expectText string
	}

	tests := []testCase{
		{
			name: "normal",
			aicall: &aicall.AIcall{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("02732972-96f1-4c51-9f76-38b32377493c"),
				},
				ReferenceType:     aicall.ReferenceTypeCall,
				AIID:              uuid.FromStringOrNil("0f7a3d29-fdb5-41ba-8fa9-3a85e02ce17a"),
				AIEngineType:      ai.EngineTypeNone,
				Gender:            aicall.GenderFemale,
				Language:          "en-US",
				ReferenceID:       uuid.FromStringOrNil("5d93dd9c-8306-11f0-8abe-23fa7bf3b155"),
				TTSStreamingPodID: "5dc667bc-8306-11f0-83b7-1fa6cc24ea64",
				TTSStreamingID:    uuid.FromStringOrNil("5def9556-8306-11f0-a4c9-b7d814746d11"),
			},
			text: "hi",

			responseMessage: &message.Message{
				Role:    message.RoleAssistant, // Changed to assistant since the ai responds
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
			mockAI := aihandler.NewMockAIHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			// Set up expectations for the mocks. Make sure arguments match what you're passing.
			mockReq.EXPECT().TTSV1StreamingSayStop(ctx, tt.aicall.TTSStreamingPodID, tt.aicall.TTSStreamingID).Return(nil)
			mockMessage.EXPECT().StreamingSend(ctx, tt.aicall.ID, tt.expectRole, tt.text, true).Return(tt.responseMessage, nil)

			if errChat := h.ChatMessage(ctx, tt.aicall, tt.text); errChat != nil {
				t.Errorf("ChatMessage() error = %v", errChat)
			}
		})
	}
}

func Test_ChatInit(t *testing.T) {

	tests := []struct {
		name   string
		ai     *ai.AI
		aicall *aicall.AIcall

		responseInitPrompt string
		responseMessage    *message.Message

		expectVariables  map[string]string
		expectInitPrompt string
	}{
		{
			name: "normal",
			ai: &ai.AI{
				Identity: identity.Identity{
					ID: uuid.FromStringOrNil("a6d0f872-f7e8-11ef-a1fa-8b9babb2a9f5"),
				},
				InitPrompt: "test message",
			},
			aicall: &aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("a5f77d7c-f7e8-11ef-a28e-3babccbf3e47"),
					CustomerID: uuid.FromStringOrNil("a64db8b8-f7e8-11ef-9a8e-f3357aace0ff"),
				},
				AIID:              uuid.FromStringOrNil("a6d0f872-f7e8-11ef-a1fa-8b9babb2a9f5"),
				ActiveflowID:      uuid.FromStringOrNil("a6aec4b4-f7e8-11ef-9b61-37d5c56e8086"),
				ReferenceType:     aicall.ReferenceTypeCall,
				ReferenceID:       uuid.FromStringOrNil("a6830630-f7e8-11ef-9fc4-7fd9341c5fe5"),
				ConfbridgeID:      uuid.FromStringOrNil("333d4aea-f7e9-11ef-873e-efd62602ccad"),
				AIEngineModel:     ai.EngineModelOpenaiGPT4,
				Gender:            aicall.GenderNuetral,
				Language:          "en-US",
				TTSStreamingPodID: "3bf63b4e-8306-11f0-8ca6-23223b21657b",
				TTSStreamingID:    uuid.FromStringOrNil("3c237ec4-8306-11f0-ab5d-b77da11dec7c"),
			},

			responseInitPrompt: "test init prompt",
			responseMessage: &message.Message{
				Content: "Hello, The lame person is the culprit",
			},

			expectVariables: map[string]string{
				variableAIcallID:      "a5f77d7c-f7e8-11ef-a28e-3babccbf3e47",
				variableAIID:          "a6d0f872-f7e8-11ef-a1fa-8b9babb2a9f5",
				variableAIEngineModel: string(ai.EngineModelOpenaiGPT4),
				variableConfbridgeID:  "333d4aea-f7e9-11ef-873e-efd62602ccad",
				variableGender:        string(aicall.GenderNuetral),
				variableLanguage:      "en-US",
			},
			expectInitPrompt: "test init prompt",
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
			mockAI := aihandler.NewMockAIHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.aicall.ActiveflowID, tt.expectVariables).Return(nil)
			mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.aicall.ActiveflowID, tt.ai.InitPrompt).Return(tt.responseInitPrompt, nil)

			if tt.aicall.ReferenceType == aicall.ReferenceTypeCall {
				mockMessage.EXPECT().StreamingSend(ctx, tt.aicall.ID, message.RoleSystem, tt.expectInitPrompt, true).Return(tt.responseMessage, nil)
			} else {
				mockMessage.EXPECT().Send(ctx, tt.aicall.ID, message.RoleSystem, tt.expectInitPrompt, true).Return(tt.responseMessage, nil)
			}

			err := h.chatInit(ctx, tt.ai, tt.aicall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_ChatInit_without_activeflow_id(t *testing.T) {

	tests := []struct {
		name   string
		ai     *ai.AI
		aicall *aicall.AIcall

		responseMessage *message.Message
	}{
		{
			name: "normal",
			ai: &ai.AI{
				EngineType: ai.EngineTypeNone,
				InitPrompt: "test message",
			},
			aicall: &aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("9bb7079c-f556-11ed-afbb-0f109793414b"),
					CustomerID: uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426614174000"),
				},
				ReferenceType:     aicall.ReferenceTypeNone,
				ReferenceID:       uuid.FromStringOrNil("55667788-9900-1122-3344-aabbccddeef1"),
				Gender:            aicall.GenderNuetral,
				Language:          "en-US",
				TTSStreamingPodID: "c19462e0-8305-11f0-b617-c33d6dbf1d54",
				TTSStreamingID:    uuid.FromStringOrNil("c1f06f4a-8305-11f0-a32c-9f4c73eee550"),
			},

			responseMessage: &message.Message{
				Content: "Hello, Bruce Willis is a ghost.",
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
			mockAI := aihandler.NewMockAIHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			mockMessage.EXPECT().Send(ctx, tt.aicall.ID, message.RoleSystem, tt.ai.InitPrompt, true).Return(tt.responseMessage, nil)

			err := h.chatInit(ctx, tt.ai, tt.aicall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_chatMessageReferenceTypeCall(t *testing.T) {

	tests := []struct {
		name           string
		aicall         *aicall.AIcall
		messageContent string

		responseMessage *message.Message
	}{
		{
			name: "normal",
			aicall: &aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("47ea05dc-ef4c-11ef-8318-af1841553e05"),
					CustomerID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
				ReferenceType:     aicall.ReferenceTypeCall,
				AIID:              uuid.FromStringOrNil("48445be0-ef4c-11ef-9ac6-f39d9cadacfd"),
				AIEngineType:      ai.EngineTypeNone,
				Gender:            aicall.GenderFemale,
				Language:          "en-US",
				ReferenceID:       uuid.FromStringOrNil("55667788-9900-1122-3344-aabbccddeef1"),
				TTSStreamingPodID: "7088be06-8304-11f0-9297-67f7f030ee2f",
				TTSStreamingID:    uuid.FromStringOrNil("70b65078-8304-11f0-9405-b397054d69e2"),
			},
			messageContent: "hi",

			responseMessage: &message.Message{
				Content: "Hello, my name is chat-gpt.",
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
			mockAI := aihandler.NewMockAIHandler(mc)
			mockMessage := messagehandler.NewMockMessageHandler(mc)

			h := &aicallHandler{
				utilHandler:    mockUtil,
				reqHandler:     mockReq,
				notifyHandler:  mockNotify,
				db:             mockDB,
				aiHandler:      mockAI,
				messageHandler: mockMessage,
			}
			ctx := context.Background()

			mockReq.EXPECT().TTSV1StreamingSayStop(ctx, tt.aicall.TTSStreamingPodID, tt.aicall.TTSStreamingID).Return(nil)
			mockMessage.EXPECT().StreamingSend(ctx, tt.aicall.ID, message.RoleUser, tt.messageContent, true).Return(tt.responseMessage, nil)

			errChat := h.chatMessageReferenceTypeCall(ctx, tt.aicall, tt.messageContent)
			if errChat != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errChat)
			}
		})
	}
}

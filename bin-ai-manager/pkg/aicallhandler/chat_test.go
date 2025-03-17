package aicallhandler

import (
	"context"
	"encoding/json"
	cmconfbridge "monorepo/bin-call-manager/models/confbridge"
	fmaction "monorepo/bin-flow-manager/models/action"
	fmactiveflow "monorepo/bin-flow-manager/models/activeflow"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
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
				ReferenceType: aicall.ReferenceTypeCall,
				AIID:          uuid.FromStringOrNil("0f7a3d29-fdb5-41ba-8fa9-3a85e02ce17a"),
				AIEngineType:  ai.EngineTypeNone,
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
				ReferenceID:   uuid.FromStringOrNil("some-reference-id"), // Add this
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

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}
			ctx := context.Background()

			// Set up expectations for the mocks. Make sure arguments match what you're passing.
			mockReq.EXPECT().CallV1CallMediaStop(ctx, tt.aicall.ReferenceID).Return(nil)
			mockReq.EXPECT().AIV1MessageSend(ctx, tt.aicall.ID, tt.expectRole, tt.text, gomock.Any()).Return(tt.responseMessage, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.aicall.ReferenceID, tt.expectText, string(tt.aicall.Gender), tt.aicall.Language, 10000).Return(nil)

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
				AIID:          uuid.FromStringOrNil("a6d0f872-f7e8-11ef-a1fa-8b9babb2a9f5"),
				ActiveflowID:  uuid.FromStringOrNil("a6aec4b4-f7e8-11ef-9b61-37d5c56e8086"),
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("a6830630-f7e8-11ef-9fc4-7fd9341c5fe5"),
				ConfbridgeID:  uuid.FromStringOrNil("333d4aea-f7e9-11ef-873e-efd62602ccad"),
				AIEngineModel: ai.EngineModelOpenaiGPT4,
				Gender:        aicall.GenderNuetral,
				Language:      "en-US",
			},

			responseInitPrompt: "test init prompt",

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

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.aicall.ActiveflowID, tt.expectVariables).Return(nil)
			mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.aicall.ActiveflowID, tt.ai.InitPrompt).Return(tt.responseInitPrompt, nil)

			mockReq.EXPECT().AIV1MessageSend(ctx, tt.aicall.Identity.ID, message.RoleSystem, tt.expectInitPrompt, gomock.Any()).Return(&message.Message{Content: "test assist"}, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.aicall.ReferenceID, "test assist", string(tt.aicall.Gender), tt.aicall.Language, 10000).Return(nil)

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
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("55667788-9900-1122-3344-aabbccddeef1"),
				Gender:        aicall.GenderNuetral,
				Language:      "en-US",
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

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}
			ctx := context.Background()

			mockReq.EXPECT().AIV1MessageSend(ctx, tt.aicall.Identity.ID, message.RoleSystem, tt.ai.InitPrompt, gomock.Any()).Return(&message.Message{Content: "test assist"}, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.aicall.ReferenceID, "test assist", string(tt.aicall.Gender), tt.aicall.Language, 10000).Return(nil)

			err := h.chatInit(ctx, tt.ai, tt.aicall)
			if err != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", err)
			}
		})
	}
}

func Test_chatMessageActionsHandle(t *testing.T) {

	tests := []struct {
		name string

		aicall  *aicall.AIcall
		actions []fmaction.Action
	}{
		{
			name: "normal",

			aicall: &aicall.AIcall{
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
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1ActiveflowPushActions(ctx, tt.aicall.ActiveflowID, tt.actions).Return(&fmactiveflow.Activeflow{}, nil)
			mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, tt.aicall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)

			if errHandle := h.chatMessageActionsHandle(ctx, tt.aicall, tt.actions); errHandle != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errHandle)
			}
		})
	}
}

func Test_chatMessageReferenceTypeCall(t *testing.T) {

	tests := []struct {
		name           string
		aicall         *aicall.AIcall
		messageContent string
	}{
		{
			name: "normal",
			aicall: &aicall.AIcall{
				Identity: identity.Identity{
					ID:         uuid.FromStringOrNil("47ea05dc-ef4c-11ef-8318-af1841553e05"),
					CustomerID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
				},
				ReferenceType: aicall.ReferenceTypeCall,
				AIID:          uuid.FromStringOrNil("48445be0-ef4c-11ef-9ac6-f39d9cadacfd"),
				AIEngineType:  ai.EngineTypeNone,
				Gender:        aicall.GenderFemale,
				Language:      "en-US",
				ReferenceID:   uuid.FromStringOrNil("55667788-9900-1122-3344-aabbccddeef1"),
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
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallMediaStop(ctx, tt.aicall.ReferenceID).Return(nil)
			mockReq.EXPECT().AIV1MessageSend(ctx, tt.aicall.Identity.ID, message.RoleUser, tt.messageContent, gomock.Any()).Return(&message.Message{Content: "test assist"}, nil)
			mockReq.EXPECT().CallV1CallTalk(ctx, tt.aicall.ReferenceID, "test assist", string(tt.aicall.Gender), tt.aicall.Language, 10000).Return(nil)

			errChat := h.chatMessageReferenceTypeCall(ctx, tt.aicall, tt.messageContent)
			if errChat != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errChat)
			}
		})
	}
}

func Test_chatMessageHandle_actions(t *testing.T) {
	tests := []struct {
		name    string
		aicall  *aicall.AIcall
		message *message.Message
	}{
		{
			name: "Action_Message",
			aicall: &aicall.AIcall{
				ActiveflowID:  uuid.FromStringOrNil("456789ab-1234-6543-3456-89abcdef0124"),
				ConfbridgeID:  uuid.FromStringOrNil("11223344-5566-7788-9900-aabbccddeef2"),
				ReferenceType: aicall.ReferenceTypeCall,
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
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}
			ctx := context.Background()

			var tmpActions []fmaction.Action
			errUnmarshal := json.Unmarshal([]byte(tt.message.Content), &tmpActions)

			if errUnmarshal == nil {
				mockReq.EXPECT().FlowV1ActiveflowPushActions(ctx, tt.aicall.ActiveflowID, gomock.Any()).Return(&fmactiveflow.Activeflow{}, nil)
				mockReq.EXPECT().CallV1ConfbridgeTerminate(ctx, tt.aicall.ConfbridgeID).Return(&cmconfbridge.Confbridge{}, nil)
			}

			errChat := h.chatMessageHandle(ctx, tt.aicall, tt.message)
			if errChat != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errChat)
			}
		})
	}
}

func Test_chatMessageHandle_text(t *testing.T) {
	tests := []struct {
		name    string
		aicall  *aicall.AIcall
		message *message.Message
	}{
		{
			name: "Text_Message",
			aicall: &aicall.AIcall{
				ReferenceType: aicall.ReferenceTypeCall,
				ReferenceID:   uuid.FromStringOrNil("55667788-9900-1122-3344-aabbccddeef1"),
				Gender:        aicall.GenderNuetral,
				Language:      "ja-JP",
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
			mockAI := aihandler.NewMockAIHandler(mc)

			h := &aicallHandler{
				utilHandler:   mockUtil,
				reqHandler:    mockReq,
				notifyHandler: mockNotify,
				db:            mockDB,
				aiHandler:     mockAI,
			}
			ctx := context.Background()

			mockReq.EXPECT().CallV1CallTalk(ctx, tt.aicall.ReferenceID, tt.message.Content, string(tt.aicall.Gender), tt.aicall.Language, 10000).Return(nil)

			errChat := h.chatMessageHandle(ctx, tt.aicall, tt.message)
			if errChat != nil {
				t.Errorf("Wrong match. expect: ok, got: %v", errChat)
			}
		})
	}
}

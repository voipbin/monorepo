package aicallhandler

import (
	"context"
	"encoding/json"
	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/pkg/aihandler"
	"monorepo/bin-ai-manager/pkg/dbhandler"
	"monorepo/bin-ai-manager/pkg/messagehandler"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"
	reflect "reflect"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

// func Test_ChatMessage(t *testing.T) {

// 	type testCase struct {
// 		name string

// 		aicall *aicall.AIcall
// 		text   string

// 		responseMessage *message.Message

// 		expectRole message.Role
// 		expectText string
// 	}

// 	tests := []testCase{
// 		{
// 			name: "normal",
// 			aicall: &aicall.AIcall{
// 				Identity: identity.Identity{
// 					ID: uuid.FromStringOrNil("02732972-96f1-4c51-9f76-38b32377493c"),
// 				},
// 				ReferenceType:     aicall.ReferenceTypeCall,
// 				AIID:              uuid.FromStringOrNil("0f7a3d29-fdb5-41ba-8fa9-3a85e02ce17a"),
// 				AIEngineType:      ai.EngineTypeNone,
// 				Gender:            aicall.GenderFemale,
// 				Language:          "en-US",
// 				ReferenceID:       uuid.FromStringOrNil("5d93dd9c-8306-11f0-8abe-23fa7bf3b155"),
// 				TTSStreamingPodID: "5dc667bc-8306-11f0-83b7-1fa6cc24ea64",
// 				TTSStreamingID:    uuid.FromStringOrNil("5def9556-8306-11f0-a4c9-b7d814746d11"),
// 			},
// 			text: "hi",

// 			responseMessage: &message.Message{
// 				Role:    message.RoleAssistant, // Changed to assistant since the ai responds
// 				Content: "Hello, my name is chat-gpt.",
// 			},

// 			expectRole: message.RoleUser,
// 			expectText: "Hello, my name is chat-gpt.",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockAI := aihandler.NewMockAIHandler(mc)
// 			mockMessage := messagehandler.NewMockMessageHandler(mc)

// 			h := &aicallHandler{
// 				utilHandler:    mockUtil,
// 				reqHandler:     mockReq,
// 				notifyHandler:  mockNotify,
// 				db:             mockDB,
// 				aiHandler:      mockAI,
// 				messageHandler: mockMessage,
// 			}
// 			ctx := context.Background()

// 			// Set up expectations for the mocks. Make sure arguments match what you're passing.
// 			mockReq.EXPECT().TTSV1StreamingSayStop(ctx, tt.aicall.TTSStreamingPodID, tt.aicall.TTSStreamingID).Return(nil)
// 			mockMessage.EXPECT().StreamingSend(ctx, tt.aicall.ID, tt.expectRole, tt.text).Return(tt.responseMessage, nil)

// 			if errChat := h.ChatMessage(ctx, tt.aicall, tt.text); errChat != nil {
// 				t.Errorf("ChatMessage() error = %v", errChat)
// 			}
// 		})
// 	}
// }

// func Test_ChatInit(t *testing.T) {

// 	tests := []struct {
// 		name   string
// 		ai     *ai.AI
// 		aicall *aicall.AIcall

// 		responseInitPrompt string
// 		responseMessage    *message.Message

// 		expectVariables  map[string]string
// 		expectInitPrompt string
// 	}{
// 		{
// 			name: "normal",
// 			ai: &ai.AI{
// 				Identity: identity.Identity{
// 					ID: uuid.FromStringOrNil("a6d0f872-f7e8-11ef-a1fa-8b9babb2a9f5"),
// 				},
// 				InitPrompt: "test message",
// 			},
// 			aicall: &aicall.AIcall{
// 				Identity: identity.Identity{
// 					ID:         uuid.FromStringOrNil("a5f77d7c-f7e8-11ef-a28e-3babccbf3e47"),
// 					CustomerID: uuid.FromStringOrNil("a64db8b8-f7e8-11ef-9a8e-f3357aace0ff"),
// 				},
// 				AIID:          uuid.FromStringOrNil("a6d0f872-f7e8-11ef-a1fa-8b9babb2a9f5"),
// 				ActiveflowID:  uuid.FromStringOrNil("a6aec4b4-f7e8-11ef-9b61-37d5c56e8086"),
// 				ReferenceType: aicall.ReferenceTypeCall,
// 				ReferenceID:   uuid.FromStringOrNil("a6830630-f7e8-11ef-9fc4-7fd9341c5fe5"),
// 				ConfbridgeID:  uuid.FromStringOrNil("333d4aea-f7e9-11ef-873e-efd62602ccad"),
// 				AIEngineModel: ai.EngineModelOpenaiGPT4,
// 				Gender:        aicall.GenderNuetral,
// 				Language:      "en-US",
// 			},

// 			responseInitPrompt: "test init prompt",
// 			responseMessage: &message.Message{
// 				Content: "Hello, The lame person is the culprit",
// 			},

// 			expectVariables: map[string]string{
// 				variableAIcallID:      "a5f77d7c-f7e8-11ef-a28e-3babccbf3e47",
// 				variableAIID:          "a6d0f872-f7e8-11ef-a1fa-8b9babb2a9f5",
// 				variableAIEngineModel: string(ai.EngineModelOpenaiGPT4),
// 				variableConfbridgeID:  "333d4aea-f7e9-11ef-873e-efd62602ccad",
// 				variableGender:        string(aicall.GenderNuetral),
// 				variableLanguage:      "en-US",
// 			},
// 			expectInitPrompt: "test init prompt",
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockAI := aihandler.NewMockAIHandler(mc)
// 			mockMessage := messagehandler.NewMockMessageHandler(mc)

// 			h := &aicallHandler{
// 				utilHandler:    mockUtil,
// 				reqHandler:     mockReq,
// 				notifyHandler:  mockNotify,
// 				db:             mockDB,
// 				aiHandler:      mockAI,
// 				messageHandler: mockMessage,
// 			}
// 			ctx := context.Background()

// 			mockReq.EXPECT().FlowV1VariableSetVariable(ctx, tt.aicall.ActiveflowID, tt.expectVariables).Return(nil)
// 			mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.aicall.ActiveflowID, tt.ai.InitPrompt).Return(tt.responseInitPrompt, nil)

// 			// if tt.aicall.ReferenceType == aicall.ReferenceTypeCall {
// 			// 	mockMessage.EXPECT().StreamingSend(ctx, tt.aicall.ID, message.RoleSystem, tt.expectInitPrompt).Return(tt.responseMessage, nil)
// 			// } else {
// 			// 	mockMessage.EXPECT().Send(ctx, tt.aicall.ID, message.RoleSystem, tt.expectInitPrompt, true).Return(tt.responseMessage, nil)
// 			// }

// 			err := h.chatInit(ctx, tt.ai, tt.aicall)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }

// func Test_ChatInit_without_activeflow_id(t *testing.T) {

// 	tests := []struct {
// 		name   string
// 		ai     *ai.AI
// 		aicall *aicall.AIcall

// 		responseMessage *message.Message
// 	}{
// 		{
// 			name: "normal",
// 			ai: &ai.AI{
// 				EngineType: ai.EngineTypeNone,
// 				InitPrompt: "test message",
// 			},
// 			aicall: &aicall.AIcall{
// 				Identity: identity.Identity{
// 					ID:         uuid.FromStringOrNil("9bb7079c-f556-11ed-afbb-0f109793414b"),
// 					CustomerID: uuid.FromStringOrNil("123e4567-e89b-12d3-a456-426614174000"),
// 				},
// 				ReferenceType: aicall.ReferenceTypeNone,
// 				ReferenceID:   uuid.FromStringOrNil("55667788-9900-1122-3344-aabbccddeef1"),
// 				Gender:        aicall.GenderNuetral,
// 				Language:      "en-US",
// 			},

// 			responseMessage: &message.Message{
// 				Content: "Hello, Bruce Willis is a ghost.",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockAI := aihandler.NewMockAIHandler(mc)
// 			mockMessage := messagehandler.NewMockMessageHandler(mc)

// 			h := &aicallHandler{
// 				utilHandler:    mockUtil,
// 				reqHandler:     mockReq,
// 				notifyHandler:  mockNotify,
// 				db:             mockDB,
// 				aiHandler:      mockAI,
// 				messageHandler: mockMessage,
// 			}
// 			ctx := context.Background()

// 			mockMessage.EXPECT().Send(ctx, tt.aicall.ID, message.RoleSystem, tt.ai.InitPrompt, true).Return(tt.responseMessage, nil)

// 			err := h.chatInit(ctx, tt.ai, tt.aicall)
// 			if err != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", err)
// 			}
// 		})
// 	}
// }

// func Test_chatMessageReferenceTypeCall(t *testing.T) {

// 	tests := []struct {
// 		name           string
// 		aicall         *aicall.AIcall
// 		messageContent string

// 		responseMessage *message.Message
// 	}{
// 		{
// 			name: "normal",
// 			aicall: &aicall.AIcall{
// 				Identity: identity.Identity{
// 					ID:         uuid.FromStringOrNil("47ea05dc-ef4c-11ef-8318-af1841553e05"),
// 					CustomerID: uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001"),
// 				},
// 				ReferenceType:     aicall.ReferenceTypeCall,
// 				AIID:              uuid.FromStringOrNil("48445be0-ef4c-11ef-9ac6-f39d9cadacfd"),
// 				AIEngineType:      ai.EngineTypeNone,
// 				Gender:            aicall.GenderFemale,
// 				Language:          "en-US",
// 				ReferenceID:       uuid.FromStringOrNil("55667788-9900-1122-3344-aabbccddeef1"),
//  			},
// 			messageContent: "hi",

// 			responseMessage: &message.Message{
// 				Content: "Hello, my name is chat-gpt.",
// 			},
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockAI := aihandler.NewMockAIHandler(mc)
// 			mockMessage := messagehandler.NewMockMessageHandler(mc)

// 			h := &aicallHandler{
// 				utilHandler:    mockUtil,
// 				reqHandler:     mockReq,
// 				notifyHandler:  mockNotify,
// 				db:             mockDB,
// 				aiHandler:      mockAI,
// 				messageHandler: mockMessage,
// 			}
// 			ctx := context.Background()

// 			mockReq.EXPECT().TTSV1StreamingSayStop(ctx, tt.aicall.TTSStreamingPodID, tt.aicall.TTSStreamingID).Return(nil)
// 			mockMessage.EXPECT().StreamingSend(ctx, tt.aicall.ID, message.RoleUser, tt.messageContent).Return(tt.responseMessage, nil)

// 			errChat := h.chatMessageReferenceTypeCall(ctx, tt.aicall, tt.messageContent)
// 			if errChat != nil {
// 				t.Errorf("Wrong match. expect: ok, got: %v", errChat)
// 			}
// 		})
// 	}
// }

// func Test_getEngineData(t *testing.T) {

// 	tests := []struct {
// 		name string

// 		ai           *ai.AI
// 		activeflowID uuid.UUID

// 		responseSubstitutes []string
// 		expectedRes         string
// 	}{
// 		{
// 			name: "normal",

// 			ai: &ai.AI{
// 				EngineData: map[string]any{
// 					"key1": "value1",
// 					"key2": 2,
// 					"key3": true,
// 					"key4": "The culprit is {${lame_person}}.",
// 					"key5": map[string]any{
// 						"subkey1": "subvalue1",
// 						"subkey2": 3,
// 						"subkey3": "The ghost is {${ghost_person}}.",
// 						"subkey4": []string{
// 							"sub list val 1",
// 							"The secret is {${secret_info}}.",
// 						},
// 					},
// 					"key6": []string{
// 						"list val 1",
// 						"The answer is {${answer_info}}.",
// 					},
// 					"key7": 4.5,
// 					"key8": nil,
// 				},
// 			},
// 			activeflowID: uuid.FromStringOrNil("d48b2510-c035-11f0-b454-83d837506895"),

// 			responseSubstitutes: []string{
// 				"response 1",
// 				"response 2",
// 				"response 3",
// 				"response 4",
// 				"response 5",
// 				"response 6",
// 				"response 7",
// 				"response 8",
// 			},
// 			// expectedRes: "",
// 			expectedRes: `{"key1":"response 1","key2":"2","key3":"true","key4":"response 2","key5":"{\"subkey1\":\"response 3\",\"subkey2\":\"3\",\"subkey3\":\"response 4\",\"subkey4\":\"[\\\"response 5\\\",\\\"response 6\\\"]\"}"}, got: {"key2":"2","key3":"true","key7":"4.5","key8":""}`,
// 		},
// 		{
// 			name: "normal",

// 			ai: &ai.AI{
// 				EngineData: map[string]any{
// 					"key2": 2,
// 					"key3": true,
// 					"key7": 4.5,
// 					"key8": nil,
// 				},
// 			},
// 			activeflowID: uuid.FromStringOrNil("d48b2510-c035-11f0-b454-83d837506895"),

// 			responseSubstitutes: []string{},
// 			// expectedRes: "",
// 			expectedRes: `{"key1":"response 1","key2":"2","key3":"true","key4":"response 2","key5":"{\"subkey1\":\"response 3\",\"subkey2\":\"3\",\"subkey3\":\"response 4\",\"subkey4\":\"[\\\"response 5\\\",\\\"response 6\\\"]\"}"}`,
// 		},
// 		{
// 			name: "sub keys",

// 			ai: &ai.AI{
// 				EngineData: map[string]any{
// 					"key1": 2,
// 					"key2": true,
// 					"key3": map[string]any{
// 						"subkey1": 4.5,
// 						"subkey2": 3,
// 						"subkey3": true,
// 						"subkey4": []string{
// 							"sub list val 1",
// 						},
// 						"subkey5": nil,
// 					},
// 				},
// 			},
// 			activeflowID: uuid.FromStringOrNil("d48b2510-c035-11f0-b454-83d837506895"),

// 			responseSubstitutes: []string{
// 				"response 1",
// 			},
// 			// expectedRes: "",
// 			expectedRes: `{"key1":"response 1","key2":"2","key3":"true","key4":"response 2","key5":"{\"subkey1\":\"response 3\",\"subkey2\":\"3\",\"subkey3\":\"response 4\",\"subkey4\":\"[\\\"response 5\\\",\\\"response 6\\\"]\"}"}`,
// 		},
// 	}

// 	for _, tt := range tests {
// 		t.Run(tt.name, func(t *testing.T) {
// 			mc := gomock.NewController(t)
// 			defer mc.Finish()

// 			mockUtil := utilhandler.NewMockUtilHandler(mc)
// 			mockReq := requesthandler.NewMockRequestHandler(mc)
// 			mockNotify := notifyhandler.NewMockNotifyHandler(mc)
// 			mockDB := dbhandler.NewMockDBHandler(mc)
// 			mockAI := aihandler.NewMockAIHandler(mc)
// 			mockMessage := messagehandler.NewMockMessageHandler(mc)

// 			h := &aicallHandler{
// 				utilHandler:    mockUtil,
// 				reqHandler:     mockReq,
// 				notifyHandler:  mockNotify,
// 				db:             mockDB,
// 				aiHandler:      mockAI,
// 				messageHandler: mockMessage,
// 			}
// 			ctx := context.Background()

// 			for _, sub := range tt.responseSubstitutes {
// 				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, gomock.Any()).Return(sub, nil)
// 			}

// 			res := h.getEngineData(ctx, tt.ai, tt.activeflowID)
// 			if res != tt.expectedRes {
// 				t.Errorf("Wrong match. expect: %v, got: %v", tt.expectedRes, res)
// 			}
// 		})
// 	}
// }

func Test_getEngineData(t *testing.T) {
	tests := []struct {
		name string

		ai           *ai.AI
		activeflowID uuid.UUID

		responseSubstitutes map[string]string
		expectedRes         string
	}{
		{
			name: "nested structure with variable substitution",
			ai: &ai.AI{
				EngineData: map[string]any{
					"key1": "value1",
					"key2": 2,
					"key3": true,
					"key4": "The culprit is {${lame_person}}.",
					"key5": map[string]any{
						"subkey1": "subvalue1",
						"subkey2": 3,
						"subkey3": "The ghost is {${ghost_person}}.",
						"subkey4": []string{
							"sub list val 1",
							"The secret is {${secret_info}}.",
						},
					},
					"key6": []string{
						"list val 1",
						"The answer is {${answer_info}}.",
					},
					"key7": 4.5,
					"key8": nil,
				},
			},
			activeflowID: uuid.FromStringOrNil("d48b2510-c035-11f0-b454-83d837506895"),
			responseSubstitutes: map[string]string{
				"value1":                           "response 1",
				"The culprit is {${lame_person}}.": "response 2",
				"subvalue1":                        "response 3",
				"The ghost is {${ghost_person}}.":  "response 4",
				"sub list val 1":                   "response 5",
				"The secret is {${secret_info}}.":  "response 6",
				"list val 1":                       "response 7",
				"The answer is {${answer_info}}.":  "response 8",
			},
			expectedRes: `{"key1":"response 1","key2":"2","key3":"true","key4":"response 2","key5":"{\"subkey1\":\"response 3\",\"subkey2\":\"3\",\"subkey3\":\"response 4\",\"subkey4\":\"[\\\"response 5\\\",\\\"response 6\\\"]\"}","key6":"[\"response 7\",\"response 8\"]","key7":"4.5","key8":""}`,
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

			for k, v := range tt.responseSubstitutes {
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, k).Return(v, nil)
			}

			res := h.getEngineData(ctx, tt.ai, tt.activeflowID)

			var expected, actual map[string]any
			if err := json.Unmarshal([]byte(tt.expectedRes), &expected); err != nil {
				t.Fatalf("invalid expectedRes JSON: %v", err)
			}
			if err := json.Unmarshal([]byte(res), &actual); err != nil {
				t.Fatalf("invalid result JSON: %v", err)
			}

			if !reflect.DeepEqual(expected, actual) {
				t.Errorf("Wrong match.\nexpected: %v\ngot: %v", expected, actual)
			}
		})
	}
}

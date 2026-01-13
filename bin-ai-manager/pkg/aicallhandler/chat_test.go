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
			expectedRes: `
{
    "key1": "response 1",
    "key2": 2,
    "key3": true,
    "key4": "response 2",
    "key5": {
        "subkey1": "response 3",
        "subkey2": 3,
        "subkey3": "response 4",
        "subkey4": [
            "response 5",
            "response 6"
        ]
    },
    "key6": [
        "response 7",
        "response 8"
    ],
    "key7": 4.5,
    "key8": null
}`,
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
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, k.Return(v, nil)
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

func Test_getEngineDataValue(t *testing.T) {
	tests := []struct {
		name string

		input          any
		activeflowID   uuid.UUID
		substitutes    map[string]string
		expectedResult any
	}{
		{
			name:         "simple string substitution",
			input:        "value1",
			activeflowID: uuid.FromStringOrNil("11111111-1111-1111-1111-111111111111"),
			substitutes: map[string]string{
				"value1": "response 1",
			},
			expectedResult: "response 1",
		},
		{
			name: "nested map with substitutions",
			input: map[string]any{
				"key1": "value1",
				"key2": 2,
				"key3": "value2",
			},
			activeflowID: uuid.FromStringOrNil("22222222-2222-2222-2222-222222222222"),
			substitutes: map[string]string{
				"value1": "response 1",
				"value2": "response 2",
			},
			expectedResult: map[string]any{
				"key1": "response 1",
				"key2": int64(2),
				"key3": "response 2",
			},
		},
		{
			name: "slice with substitutions",
			input: []any{
				"value1",
				"value2",
				3,
			},
			activeflowID: uuid.FromStringOrNil("33333333-3333-3333-3333-333333333333"),
			substitutes: map[string]string{
				"value1": "response 1",
				"value2": "response 2",
			},
			expectedResult: []any{
				"response 1",
				"response 2",
				int64(3),
			},
		},
		{
			name:           "nil value",
			input:          nil,
			activeflowID:   uuid.FromStringOrNil("44444444-4444-4444-4444-444444444444"),
			substitutes:    map[string]string{},
			expectedResult: nil,
		},
		{
			name: "mixed nested structure",
			input: map[string]any{
				"key1": "value1",
				"key2": []any{
					"value2",
					map[string]any{
						"sub1": "value3",
						"sub2": 10,
					},
				},
			},
			activeflowID: uuid.FromStringOrNil("55555555-5555-5555-5555-555555555555"),
			substitutes: map[string]string{
				"value1": "response 1",
				"value2": "response 2",
				"value3": "response 3",
			},
			expectedResult: map[string]any{
				"key1": "response 1",
				"key2": []any{
					"response 2",
					map[string]any{
						"sub1": "response 3",
						"sub2": int64(10),
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

			// string substitution mocking
			for k, v := range tt.substitutes {
				mockReq.EXPECT().FlowV1VariableSubstitute(ctx, tt.activeflowID, k.Return(v, nil)
			}

			actual := h.getEngineDataValue(ctx, tt.input, tt.activeflowID)

			if !reflect.DeepEqual(tt.expectedResult, actual) {
				t.Errorf("Wrong match.\nexpected: %#v\ngot: %#v", tt.expectedResult, actual)
			}
		})
	}
}

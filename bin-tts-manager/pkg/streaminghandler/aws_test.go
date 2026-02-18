package streaminghandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-tts-manager/models/streaming"
	"testing"

	"github.com/aws/aws-sdk-go-v2/service/polly/types"
	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_awsGetVoiceIDByLangGender(t *testing.T) {

	tests := []struct {
		name     string
		language string
		gender   streaming.Gender
		expected types.VoiceId
	}{
		{
			name:     "exact match: english male",
			language: "english",
			gender:   streaming.GenderMale,
			expected: types.VoiceIdMatthew,
		},
		{
			name:     "exact match: english female",
			language: "english",
			gender:   streaming.GenderFemale,
			expected: types.VoiceIdJoanna,
		},
		{
			name:     "exact match: japanese male",
			language: "japanese",
			gender:   streaming.GenderMale,
			expected: types.VoiceIdTakumi,
		},
		{
			name:     "exact match: german female",
			language: "german",
			gender:   streaming.GenderFemale,
			expected: types.VoiceIdMarlene,
		},
		{
			name:     "fallback to neutral",
			language: "french",
			gender:   streaming.GenderNeutral,
			expected: types.VoiceIdCeline,
		},
		{
			name:     "language with region code",
			language: "english_us",
			gender:   streaming.GenderMale,
			expected: types.VoiceIdMatthew,
		},
		{
			name:     "language with dash region code",
			language: "english-us",
			gender:   streaming.GenderFemale,
			expected: types.VoiceIdJoanna,
		},
		{
			name:     "case insensitivity",
			language: "JAPANESE",
			gender:   streaming.GenderFemale,
			expected: types.VoiceIdMizuki,
		},
		{
			name:     "unknown language returns empty",
			language: "klingon",
			gender:   streaming.GenderMale,
			expected: "",
		},
		{
			name:     "empty language returns empty",
			language: "",
			gender:   streaming.GenderMale,
			expected: "",
		},
		{
			name:     "portuguese female",
			language: "portuguese",
			gender:   streaming.GenderFemale,
			expected: types.VoiceIdCamila,
		},
		{
			name:     "polish male",
			language: "polish",
			gender:   streaming.GenderMale,
			expected: types.VoiceIdJacek,
		},
		{
			name:     "unknown gender falls back to neutral",
			language: "italian",
			gender:   "unknown_gender",
			expected: types.VoiceIdCarla,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &awsHandler{}
			result := h.getVoiceIDByLangGender(tt.language, tt.gender)
			if result != tt.expected {
				t.Errorf("got %s, expected %s", result, tt.expected)
			}
		})
	}
}

func Test_awsGetVoiceID(t *testing.T) {

	tests := []struct {
		name string

		streaming *streaming.Streaming

		// mock setup for flow variable lookup
		responseVariable *fmvariable.Variable
		responseErr      error

		expected string
	}{
		{
			name: "tier 1: explicit voice ID",
			streaming: &streaming.Streaming{
				VoiceID:  "Ruth",
				Language: "english",
				Gender:   streaming.GenderFemale,
			},
			expected: "Ruth",
		},
		{
			name: "tier 2: flow variable",
			streaming: &streaming.Streaming{
				ActiveflowID: uuid.FromStringOrNil("b1c2d3e4-f5a6-7890-abcd-ef1234567890"),
				Language:      "english",
				Gender:        streaming.GenderMale,
			},
			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{
					variableAWSVoiceID: "Kevin",
				},
			},
			expected: "Kevin",
		},
		{
			name: "tier 3: language+gender lookup",
			streaming: &streaming.Streaming{
				Language: "spanish",
				Gender:   streaming.GenderMale,
			},
			expected: string(types.VoiceIdEnrique),
		},
		{
			name: "tier 4: default fallback",
			streaming: &streaming.Streaming{
				Language: "klingon",
				Gender:   streaming.GenderMale,
			},
			expected: defaultAWSDefaultVoiceID,
		},
		{
			name: "tier 2 skipped when activeflow is nil",
			streaming: &streaming.Streaming{
				Language: "italian",
				Gender:   streaming.GenderFemale,
			},
			expected: string(types.VoiceIdCarla),
		},
		{
			name: "tier 2 skipped when variable not found",
			streaming: &streaming.Streaming{
				ActiveflowID: uuid.FromStringOrNil("b1c2d3e4-f5a6-7890-abcd-ef1234567890"),
				Language:      "dutch",
				Gender:        streaming.GenderMale,
			},
			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{
					"some.other.variable": "value",
				},
			},
			expected: string(types.VoiceIdRuben),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &awsHandler{
				reqHandler: mockReq,
			}
			ctx := context.Background()

			if tt.streaming.ActiveflowID != uuid.Nil {
				mockReq.EXPECT().FlowV1VariableGet(ctx, tt.streaming.ActiveflowID).Return(tt.responseVariable, tt.responseErr)
			}

			result := h.getVoiceID(ctx, tt.streaming)
			if result != tt.expected {
				t.Errorf("got %s, expected %s", result, tt.expected)
			}
		})
	}
}

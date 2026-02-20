package streaminghandler

import (
	"context"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-tts-manager/models/streaming"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_getVoiceID_getVoiceIDByLangGender(t *testing.T) {

	tests := []struct {
		name     string
		language string
		gender   streaming.Gender
		expected string
	}{
		{
			name:     "exact match: english male",
			language: "english",
			gender:   streaming.GenderMale,
			expected: "21m00Tcm4TlvDq8ikWAM",
		},
		{
			name:     "exact match: japanese female",
			language: "japanese",
			gender:   streaming.GenderFemale,
			expected: "PmgfHCGeS5b7sH90BOOJ",
		},
		{
			name:     "fallback to neutral",
			language: "french",
			gender:   streaming.GenderNeutral,
			expected: "SmWACbi37pETyxxMhSpc",
		},
		{
			name:     "language with region code, fallback works",
			language: "english_us",
			gender:   streaming.GenderFemale,
			expected: "EXAVITQu4vr4xnSDxMaL",
		},
		{
			name:     "no match at all, fallback to default",
			language: "klingon",
			gender:   streaming.GenderMale,
			expected: defaultElevenlabsVoiceID,
		},
		{
			name:     "case insensitivity",
			language: "JAPANESE",
			gender:   streaming.GenderMale,
			expected: "Mv8AjrYZCBkdsmDHNwcB",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &elevenlabsHandler{
				reqHandler: mockReq,
			}
			ctx := context.Background()

			st := &streaming.Streaming{
				Language: tt.language,
				Gender:   tt.gender,
			}

			result := h.getVoiceID(ctx, st)
			if result != tt.expected {
				t.Errorf("got %s, expected %s", result, tt.expected)
			}
		})
	}
}

func Test_getVoiceID_getVoiceIDByVariable(t *testing.T) {

	tests := []struct {
		name         string
		activeflowID uuid.UUID

		responseVariable *fmvariable.Variable

		expectedRes string
	}{
		{
			name:         "exact match: english male",
			activeflowID: uuid.FromStringOrNil("f05c4fa0-87d0-11f0-9e6c-c393f26cfebb"),

			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{
					variableElevenlabsVoiceID: "21m00Tcm4TlvDq8ikWAM",
				},
			},

			expectedRes: "21m00Tcm4TlvDq8ikWAM",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)

			h := &elevenlabsHandler{
				reqHandler: mockReq,
			}
			ctx := context.Background()

			mockReq.EXPECT().FlowV1VariableGet(ctx, tt.activeflowID).Return(tt.responseVariable, nil)

			st := &streaming.Streaming{
				ActiveflowID: tt.activeflowID,
			}

			result := h.getVoiceID(ctx, st)
			if result != tt.expectedRes {
				t.Errorf("Wrong match. got: %s, expected: %s", result, tt.expectedRes)
			}
		})
	}
}

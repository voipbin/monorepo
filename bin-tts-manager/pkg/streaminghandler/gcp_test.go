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

func Test_gcpHandler_extractLangCode(t *testing.T) {
	tests := []struct {
		name         string
		voiceID      string
		fallbackLang string
		expected     string
	}{
		{
			name:         "standard Chirp3-HD voice",
			voiceID:      "en-US-Chirp3-HD-Charon",
			fallbackLang: "",
			expected:     "en-US",
		},
		{
			name:         "Japanese voice",
			voiceID:      "ja-JP-Chirp3-HD-Aoede",
			fallbackLang: "",
			expected:     "ja-JP",
		},
		{
			name:         "Chinese voice",
			voiceID:      "cmn-CN-Chirp3-HD-Charon",
			fallbackLang: "",
			expected:     "cmn-CN",
		},
		{
			name:         "non-Chirp3 voice with fallback",
			voiceID:      "some-custom-voice",
			fallbackLang: "fr-FR",
			expected:     "some-custom-voice",
		},
		{
			name:         "empty voice ID with fallback",
			voiceID:      "",
			fallbackLang: "de-DE",
			expected:     "de-DE",
		},
		{
			name:         "empty voice ID without fallback",
			voiceID:      "",
			fallbackLang: "",
			expected:     "en-US",
		},
	}

	h := &gcpHandler{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.extractLangCode(tt.voiceID, tt.fallbackLang)
			if result != tt.expected {
				t.Errorf("got %s, expected %s", result, tt.expected)
			}
		})
	}
}

func Test_gcpHandler_getVoiceIDByLangGender(t *testing.T) {
	tests := []struct {
		name     string
		language string
		gender   streaming.Gender
		expected string
	}{
		{
			name:     "english male",
			language: "english",
			gender:   streaming.GenderMale,
			expected: "en-US-Chirp3-HD-Charon",
		},
		{
			name:     "english female",
			language: "english",
			gender:   streaming.GenderFemale,
			expected: "en-US-Chirp3-HD-Aoede",
		},
		{
			name:     "japanese male",
			language: "japanese",
			gender:   streaming.GenderMale,
			expected: "ja-JP-Chirp3-HD-Charon",
		},
		{
			name:     "french neutral fallback",
			language: "french",
			gender:   streaming.GenderNeutral,
			expected: "fr-FR-Chirp3-HD-Aoede",
		},
		{
			name:     "language with region code",
			language: "english_us",
			gender:   streaming.GenderMale,
			expected: "en-US-Chirp3-HD-Charon",
		},
		{
			name:     "case insensitive",
			language: "JAPANESE",
			gender:   streaming.GenderFemale,
			expected: "ja-JP-Chirp3-HD-Aoede",
		},
		{
			name:     "language with dash region",
			language: "english-us",
			gender:   streaming.GenderMale,
			expected: "en-US-Chirp3-HD-Charon",
		},
		{
			name:     "unknown language returns empty",
			language: "klingon",
			gender:   streaming.GenderMale,
			expected: "",
		},
	}

	h := &gcpHandler{}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := h.getVoiceIDByLangGender(tt.language, tt.gender)
			if result != tt.expected {
				t.Errorf("got %s, expected %s", result, tt.expected)
			}
		})
	}
}

func Test_gcpHandler_getVoiceID(t *testing.T) {
	tests := []struct {
		name string

		streaming *streaming.Streaming

		// mock setup
		responseVariable *fmvariable.Variable
		responseErr      error

		expected string
	}{
		{
			name: "explicit voice ID takes priority",
			streaming: &streaming.Streaming{
				VoiceID:  "custom-voice-id",
				Language: "english",
				Gender:   streaming.GenderMale,
			},
			expected: "custom-voice-id",
		},
		{
			name: "flow variable takes priority over language/gender",
			streaming: &streaming.Streaming{
				ActiveflowID: uuid.FromStringOrNil("f05c4fa0-87d0-11f0-9e6c-c393f26cfebb"),
				Language:      "english",
				Gender:        streaming.GenderMale,
			},
			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{
					variableGCPVoiceID: "variable-voice-id",
				},
			},
			expected: "variable-voice-id",
		},
		{
			name: "language/gender mapping",
			streaming: &streaming.Streaming{
				Language: "japanese",
				Gender:   streaming.GenderFemale,
			},
			expected: "ja-JP-Chirp3-HD-Aoede",
		},
		{
			name: "fallback to default",
			streaming: &streaming.Streaming{
				Language: "klingon",
				Gender:   streaming.GenderMale,
			},
			expected: defaultGCPDefaultVoiceID,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &gcpHandler{reqHandler: mockReq}
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

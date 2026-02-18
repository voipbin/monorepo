package streaminghandler

import (
	"bytes"
	"context"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-tts-manager/models/streaming"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_downsample(t *testing.T) {
	tests := []struct {
		name     string
		data     []byte
		factor   int
		expected []byte
	}{
		{
			name:     "factor 1 returns original",
			data:     []byte{0x01, 0x02, 0x03, 0x04},
			factor:   1,
			expected: []byte{0x01, 0x02, 0x03, 0x04},
		},
		{
			name:   "factor 3: 24kHz to 8kHz (6 samples -> 2)",
			data:   []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08, 0x09, 0x0A, 0x0B, 0x0C},
			factor: 3,
			expected: []byte{0x01, 0x02, 0x07, 0x08},
		},
		{
			name:   "factor 2: 16kHz to 8kHz",
			data:   []byte{0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88},
			factor: 2,
			expected: []byte{0x11, 0x22, 0x55, 0x66},
		},
		{
			name:     "empty data",
			data:     []byte{},
			factor:   3,
			expected: []byte{},
		},
		{
			name:     "single sample with factor 3",
			data:     []byte{0xAA, 0xBB},
			factor:   3,
			expected: []byte{0xAA, 0xBB},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := downsample(tt.data, tt.factor)
			if !bytes.Equal(result, tt.expected) {
				t.Errorf("got %v, expected %v", result, tt.expected)
			}
		})
	}
}

func Test_gcpGetVoiceIDByLangGender(t *testing.T) {

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
			expected: "en-US-Chirp3-HD-Charon",
		},
		{
			name:     "exact match: english female",
			language: "english",
			gender:   streaming.GenderFemale,
			expected: "en-US-Chirp3-HD-Aoede",
		},
		{
			name:     "exact match: japanese male",
			language: "japanese",
			gender:   streaming.GenderMale,
			expected: "ja-JP-Chirp3-HD-Charon",
		},
		{
			name:     "exact match: korean female",
			language: "korean",
			gender:   streaming.GenderFemale,
			expected: "ko-KR-Chirp3-HD-Aoede",
		},
		{
			name:     "fallback to neutral",
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
			name:     "language with dash region code",
			language: "english-us",
			gender:   streaming.GenderFemale,
			expected: "en-US-Chirp3-HD-Aoede",
		},
		{
			name:     "case insensitivity",
			language: "JAPANESE",
			gender:   streaming.GenderFemale,
			expected: "ja-JP-Chirp3-HD-Aoede",
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
			name:     "chinese male",
			language: "chinese",
			gender:   streaming.GenderMale,
			expected: "cmn-CN-Chirp3-HD-Charon",
		},
		{
			name:     "arabic neutral fallback",
			language: "arabic",
			gender:   "unknown_gender",
			expected: "ar-XA-Chirp3-HD-Aoede",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &gcpHandler{}
			result := h.getVoiceIDByLangGender(tt.language, tt.gender)
			if result != tt.expected {
				t.Errorf("got %s, expected %s", result, tt.expected)
			}
		})
	}
}

func Test_gcpExtractLangCode(t *testing.T) {

	tests := []struct {
		name         string
		voiceID      string
		fallbackLang string
		expected     string
	}{
		{
			name:         "standard chirp3 voice",
			voiceID:      "en-US-Chirp3-HD-Charon",
			fallbackLang: "en-US",
			expected:     "en-US",
		},
		{
			name:         "japanese chirp3 voice",
			voiceID:      "ja-JP-Chirp3-HD-Aoede",
			fallbackLang: "ja-JP",
			expected:     "ja-JP",
		},
		{
			name:         "chinese chirp3 voice",
			voiceID:      "cmn-CN-Chirp3-HD-Charon",
			fallbackLang: "cmn-CN",
			expected:     "cmn-CN",
		},
		{
			name:         "non-chirp3 voice uses fallback",
			voiceID:      "custom-voice-id",
			fallbackLang: "de-DE",
			expected:     "custom-voice-id",
		},
		{
			name:         "empty voice with fallback",
			voiceID:      "",
			fallbackLang: "ko-KR",
			expected:     "ko-KR",
		},
		{
			name:         "empty voice and empty fallback",
			voiceID:      "",
			fallbackLang: "",
			expected:     "en-US",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			h := &gcpHandler{}
			result := h.extractLangCode(tt.voiceID, tt.fallbackLang)
			if result != tt.expected {
				t.Errorf("got %s, expected %s", result, tt.expected)
			}
		})
	}
}

func Test_gcpGetVoiceID(t *testing.T) {

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
				VoiceID:  "en-US-Chirp3-HD-Fenrir",
				Language: "english",
				Gender:   streaming.GenderMale,
			},
			expected: "en-US-Chirp3-HD-Fenrir",
		},
		{
			name: "tier 2: flow variable",
			streaming: &streaming.Streaming{
				ActiveflowID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
				Language:      "english",
				Gender:        streaming.GenderMale,
			},
			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{
					variableGCPVoiceID: "ja-JP-Chirp3-HD-Kore",
				},
			},
			expected: "ja-JP-Chirp3-HD-Kore",
		},
		{
			name: "tier 3: language+gender lookup",
			streaming: &streaming.Streaming{
				Language: "german",
				Gender:   streaming.GenderFemale,
			},
			expected: "de-DE-Chirp3-HD-Aoede",
		},
		{
			name: "tier 4: default fallback",
			streaming: &streaming.Streaming{
				Language: "klingon",
				Gender:   streaming.GenderMale,
			},
			expected: defaultGCPDefaultVoiceID,
		},
		{
			name: "tier 2 skipped when activeflow is nil",
			streaming: &streaming.Streaming{
				Language: "spanish",
				Gender:   streaming.GenderMale,
			},
			expected: "es-ES-Chirp3-HD-Charon",
		},
		{
			name: "tier 2 skipped when variable not found",
			streaming: &streaming.Streaming{
				ActiveflowID: uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890"),
				Language:      "italian",
				Gender:        streaming.GenderFemale,
			},
			responseVariable: &fmvariable.Variable{
				Variables: map[string]string{
					"some.other.variable": "value",
				},
			},
			expected: "it-IT-Chirp3-HD-Aoede",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mc := gomock.NewController(t)
			defer mc.Finish()

			mockReq := requesthandler.NewMockRequestHandler(mc)
			h := &gcpHandler{
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

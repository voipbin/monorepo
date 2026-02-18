package streaminghandler

import (
	"bytes"
	"context"
	"encoding/binary"
	"monorepo/bin-common-handler/pkg/requesthandler"
	fmvariable "monorepo/bin-flow-manager/models/variable"
	"monorepo/bin-tts-manager/models/streaming"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

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

func Test_downsample24kTo16k(t *testing.T) {
	// Helper to encode a slice of int16 samples into little-endian bytes.
	encode := func(samples []int16) []byte {
		buf := make([]byte, len(samples)*2)
		for i, s := range samples {
			binary.LittleEndian.PutUint16(buf[i*2:], uint16(s))
		}
		return buf
	}

	// Helper to decode little-endian bytes into int16 samples.
	decode := func(data []byte) []int16 {
		samples := make([]int16, len(data)/2)
		for i := range samples {
			samples[i] = int16(binary.LittleEndian.Uint16(data[i*2:]))
		}
		return samples
	}

	tests := []struct {
		name     string
		input    []int16
		expected []int16
	}{
		{
			name:     "one group of 3 samples",
			input:    []int16{100, 200, 300},
			expected: []int16{100, 250}, // sample 0 copied, sample 1 = avg(200,300)
		},
		{
			name:     "two groups of 3 samples",
			input:    []int16{1000, 2000, 3000, 4000, 5000, 6000},
			expected: []int16{1000, 2500, 4000, 5500},
		},
		{
			name:     "negative samples",
			input:    []int16{-100, -200, -300},
			expected: []int16{-100, -250},
		},
		{
			name:     "fewer than 3 samples returns input unchanged",
			input:    []int16{100, 200},
			expected: []int16{100, 200},
		},
		{
			name:     "trailing samples beyond last full group are dropped",
			input:    []int16{100, 200, 300, 400},
			expected: []int16{100, 250}, // only first group of 3 processed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := downsample24kTo16k(encode(tt.input))
			got := decode(result)

			if !bytes.Equal(encode(got), encode(tt.expected)) {
				t.Errorf("got %v, expected %v", got, tt.expected)
			}
		})
	}
}

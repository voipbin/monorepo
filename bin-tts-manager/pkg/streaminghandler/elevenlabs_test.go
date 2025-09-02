package streaminghandler

import (
	"bytes"
	"context"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-tts-manager/models/streaming"
	"testing"

	"github.com/gofrs/uuid"
	gomock "go.uber.org/mock/gomock"
)

func Test_getDataSamples(t *testing.T) {
	type test struct {
		name       string
		inputRate  int
		inputData  []byte
		expectData []byte
		expectErr  bool
	}

	tests := []test{
		{
			name:       "same sample rate (8000Hz), no change",
			inputRate:  8000,
			inputData:  []byte{0x01, 0x02, 0x03, 0x04},
			expectData: []byte{0x01, 0x02, 0x03, 0x04},
			expectErr:  false,
		},
		{
			name:      "16000Hz downsample to 8000Hz",
			inputRate: 16000,
			inputData: []byte{
				0x01, 0x02, // sample 1
				0x03, 0x04, // skip
				0x05, 0x06, // sample 2
				0x07, 0x08, // skip
			},
			expectData: []byte{
				0x01, 0x02,
				0x05, 0x06,
			},
			expectErr: false,
		},
		{
			name:      "unsupported rate 11025Hz",
			inputRate: 11025,
			inputData: []byte{0x01, 0x02, 0x03, 0x04},
			expectErr: true,
		},
		{
			name:      "24000Hz downsample to 8000Hz (factor 3)",
			inputRate: 24000,
			inputData: []byte{
				0x01, 0x02, // keep
				0x03, 0x04, // skip
				0x05, 0x06, // skip
				0x07, 0x08, // keep
				0x09, 0x0A, // skip
				0x0B, 0x0C, // skip
			},
			expectData: []byte{
				0x01, 0x02,
				0x07, 0x08,
			},
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &elevenlabsHandler{}
			output, err := handler.getDataSamples(tt.inputRate, tt.inputData)

			if tt.expectErr {
				if err == nil {
					t.Errorf("expected error but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !bytes.Equal(output, tt.expectData) {
				t.Errorf("mismatched data.\nexpected: %v\ngot:      %v", tt.expectData, output)
			}
		})
	}
}

func Test_getVoiceID_with_no_elevenlabs_voice_id(t *testing.T) {

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
			expected: "yoZ06aMxZJJ28mfd3POQ",
		},
		{
			name:     "fallback to neutral",
			language: "french",
			gender:   streaming.GenderNeutral,
			expected: "EXAVITQu4vr4xnSDxMaL",
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
			expected: "21m00Tcm4TlvDq8ikWAM",
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

			result := h.getVoiceID(ctx, uuid.Nil, tt.language, tt.gender)
			if result != tt.expected {
				t.Errorf("got %s, expected %s", result, tt.expected)
			}
		})
	}
}

func Test_convertAndWrapPCMData(t *testing.T) {

	tests := []struct {
		name        string
		inputFormat string
		rawData     []byte
		expectedRes []byte
		expectError bool
	}{
		{
			name:        "valid pcm_16000 input",
			inputFormat: "pcm_16000",
			rawData:     []byte{0x11, 0x22, 0x33, 0x44}, // 2 samples
			expectedRes: []byte{
				0x11, 0x22,
			},
			expectError: false,
		},
		{
			name:        "odd length input (error)",
			inputFormat: "pcm_16000",
			rawData:     []byte{0x01},
			expectError: true,
		},
		{
			name:        "unsupported format",
			inputFormat: "mp3_44100",
			rawData:     []byte{0x00, 0x01},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handler := &elevenlabsHandler{}

			res, err := handler.convertAndWrapPCMData(tt.inputFormat, tt.rawData)
			if tt.expectError {
				if err == nil {
					t.Errorf("expected error, but got nil")
				}
				return
			}

			if err != nil {
				t.Errorf("unexpected error: %v", err)
				return
			}

			if !bytes.Equal(res, tt.expectedRes) {
				t.Errorf("unexpected output:\nExpected: %v\nGot:      %v", tt.expectedRes, res)
			}
		})
	}
}

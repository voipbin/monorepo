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
			name:       "same sample rate (16000Hz), no change",
			inputRate:  16000,
			inputData:  []byte{0x01, 0x02, 0x03, 0x04},
			expectData: []byte{0x01, 0x02, 0x03, 0x04},
			expectErr:  false,
		},
		{
			name:      "48000Hz downsample to 16000Hz (factor 3)",
			inputRate: 48000,
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
		{
			name:      "unsupported rate 11025Hz",
			inputRate: 11025,
			inputData: []byte{0x01, 0x02, 0x03, 0x04},
			expectErr: true,
		},
		{
			name:      "8000Hz cannot upsample to 16000Hz",
			inputRate: 8000,
			inputData: []byte{0x01, 0x02, 0x03, 0x04},
			expectErr: true,
		},
		{
			name:      "24000Hz non-integer factor to 16000Hz",
			inputRate: 24000,
			inputData: []byte{0x01, 0x02, 0x03, 0x04},
			expectErr: true,
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

func Test_convertAndWrapPCMData(t *testing.T) {

	tests := []struct {
		name        string
		inputFormat string
		rawData     []byte
		expectedRes []byte
		expectError bool
	}{
		{
			name:        "valid pcm_16000 input (passthrough)",
			inputFormat: "pcm_16000",
			rawData:     []byte{0x11, 0x22, 0x33, 0x44}, // 2 samples, no conversion needed
			expectedRes: []byte{0x11, 0x22, 0x33, 0x44},
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

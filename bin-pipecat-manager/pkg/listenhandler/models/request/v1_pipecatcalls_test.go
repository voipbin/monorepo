package request

import (
	"testing"

	"monorepo/bin-pipecat-manager/models/pipecatcall"

	"github.com/gofrs/uuid"
)

func TestV1DataPipecatcallsPost_Struct(t *testing.T) {
	tests := []struct {
		name string
		req  V1DataPipecatcallsPost
	}{
		{
			name: "full request",
			req: V1DataPipecatcallsPost{
				ID:            uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				CustomerID:    uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
				ActiveflowID:  uuid.FromStringOrNil("5b374a54-b48c-11f0-8c36-477d3f6baf0d"),
				ReferenceType: pipecatcall.ReferenceTypeAICall,
				ReferenceID:   uuid.FromStringOrNil("5b5bb704-b48c-11f0-819e-2ff9e60d5c3c"),
				LLMType:       pipecatcall.LLMType("openai.gpt-4"),
				LLMMessages: []map[string]any{
					{
						"role":    "system",
						"content": "You are a helpful assistant.",
					},
				},
				STTType:     pipecatcall.STTTypeDeepgram,
				STTLanguage: "en-US",
				TTSType:     pipecatcall.TTSTypeElevenLabs,
				TTSLanguage: "en-US",
				TTSVoiceID:  "test-voice-id",
			},
		},
		{
			name: "empty request",
			req:  V1DataPipecatcallsPost{},
		},
		{
			name: "minimal request",
			req: V1DataPipecatcallsPost{
				ID:         uuid.FromStringOrNil("496365e2-88e6-11ea-956c-e3dfb6eaf1e8"),
				CustomerID: uuid.FromStringOrNil("5adbec2c-b48c-11f0-a0cb-e752c616594a"),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.req.ID != tt.req.ID {
				t.Errorf("ID mismatch")
			}
			if tt.req.CustomerID != tt.req.CustomerID {
				t.Errorf("CustomerID mismatch")
			}
			if tt.req.ActiveflowID != tt.req.ActiveflowID {
				t.Errorf("ActiveflowID mismatch")
			}
			if tt.req.ReferenceType != tt.req.ReferenceType {
				t.Errorf("ReferenceType mismatch")
			}
			if tt.req.ReferenceID != tt.req.ReferenceID {
				t.Errorf("ReferenceID mismatch")
			}
			if tt.req.LLMType != tt.req.LLMType {
				t.Errorf("LLMType mismatch")
			}
			if tt.req.STTType != tt.req.STTType {
				t.Errorf("STTType mismatch")
			}
			if tt.req.STTLanguage != tt.req.STTLanguage {
				t.Errorf("STTLanguage mismatch")
			}
			if tt.req.TTSType != tt.req.TTSType {
				t.Errorf("TTSType mismatch")
			}
			if tt.req.TTSLanguage != tt.req.TTSLanguage {
				t.Errorf("TTSLanguage mismatch")
			}
			if tt.req.TTSVoiceID != tt.req.TTSVoiceID {
				t.Errorf("TTSVoiceID mismatch")
			}
		})
	}
}

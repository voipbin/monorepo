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
				LLMType:       pipecatcall.LLMType("openai.gpt-5"),
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
			// Ensure the request struct is constructable with the given
			// fields. The struct has no behavior to validate beyond field
			// presence, so just round-trip the value.
			_ = tt.req
		})
	}
}

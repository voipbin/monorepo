package audiohandler

import (
	"context"
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-tts-manager/models/tts"
)

func Test_AudioCreate_unsupportedProvider(t *testing.T) {
	h := &audioHandler{}
	ctx := context.Background()
	callID := uuid.FromStringOrNil("a1b2c3d4-e5f6-7890-abcd-ef1234567890")

	tests := []struct {
		name     string
		provider tts.Provider
	}{
		{
			name:     "empty provider returns error",
			provider: "",
		},
		{
			name:     "unknown provider returns error",
			provider: "unknown",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := h.AudioCreate(ctx, callID, "<speak>hello</speak>", "en-US", tt.provider, "", "/tmp/test.wav")
			if err == nil {
				t.Errorf("expected error for provider %q, got nil", tt.provider)
			}
		})
	}
}

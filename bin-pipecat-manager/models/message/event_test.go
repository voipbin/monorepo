package message

import "testing"

func TestEventTypes(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		want     string
	}{
		{
			name:     "bot transcription",
			constant: EventTypeBotTranscription,
			want:     "message_bot_transcription",
		},
		{
			name:     "user transcription",
			constant: EventTypeUserTranscription,
			want:     "message_user_transcription",
		},
		{
			name:     "bot llm",
			constant: EventTypeBotLLM,
			want:     "message_bot_llm",
		},
		{
			name:     "user llm",
			constant: EventTypeUserLLM,
			want:     "message_user_llm",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.want {
				t.Errorf("EventType = %v, want %v", tt.constant, tt.want)
			}
		})
	}
}

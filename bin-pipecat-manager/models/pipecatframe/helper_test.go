package pipecatframe

import (
	"testing"
)

func TestCommonFrameMessage(t *testing.T) {
	tests := []struct {
		name string

		id    string
		label string
		typ   string
		data  interface{}
	}{
		{
			name: "creates_frame_with_all_fields",

			id:    "test-id-123",
			label: "rtvi-ai",
			typ:   "bot-transcription",
			data:  map[string]string{"text": "hello"},
		},
		{
			name: "creates_frame_with_nil_data",

			id:    "test-id-456",
			label: "rtvi-ai",
			typ:   "user-started-speaking",
			data:  nil,
		},
		{
			name: "creates_frame_with_empty_fields",

			id:    "",
			label: "",
			typ:   "",
			data:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			frame := CommonFrameMessage{
				ID:    tt.id,
				Label: tt.label,
				Type:  tt.typ,
				Data:  tt.data,
			}

			if frame.ID != tt.id {
				t.Errorf("Wrong ID. expect: %s, got: %s", tt.id, frame.ID)
			}
			if frame.Label != tt.label {
				t.Errorf("Wrong Label. expect: %s, got: %s", tt.label, frame.Label)
			}
			if frame.Type != tt.typ {
				t.Errorf("Wrong Type. expect: %s, got: %s", tt.typ, frame.Type)
			}
		})
	}
}

func TestRTVIFrameTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{
			name:     "bot_transcription",
			constant: RTVIFrameTypeBotTranscription,
			expected: "bot-transcription",
		},
		{
			name:     "user_transcription",
			constant: RTVIFrameTypeUserTranscription,
			expected: "user-transcription",
		},
		{
			name:     "bot_llm_text",
			constant: RTVIFrameTypeBotLLMText,
			expected: "bot-llm-text",
		},
		{
			name:     "bot_llm_started",
			constant: RTVIFrameTypeBotLLMStarted,
			expected: "bot-llm-started",
		},
		{
			name:     "bot_llm_stopped",
			constant: RTVIFrameTypeBotLLMStopped,
			expected: "bot-llm-stopped",
		},
		{
			name:     "bot_tts_started",
			constant: RTVIFrameTypeBotTTSStarted,
			expected: "bot-tts-started",
		},
		{
			name:     "bot_tts_stopped",
			constant: RTVIFrameTypeBotTTSStopped,
			expected: "bot-tts-stopped",
		},
		{
			name:     "user_started_speaking",
			constant: RTVIFrameTypeUserStartedSpeaking,
			expected: "user-started-speaking",
		},
		{
			name:     "user_stopped_speaking",
			constant: RTVIFrameTypeUserStoppedSpeaking,
			expected: "user-stopped-speaking",
		},
		{
			name:     "bot_started_speaking",
			constant: RTVIFrameTypeBotStartedSpeaking,
			expected: "bot-started-speaking",
		},
		{
			name:     "bot_stopped_speaking",
			constant: RTVIFrameTypeBotStoppedSpeaking,
			expected: "bot-stopped-speaking",
		},
		{
			name:     "metrics",
			constant: RTVIFrameTypeMetrics,
			expected: "metrics",
		},
		{
			name:     "user_llm_text",
			constant: RTVIFrameTypeUserLLMText,
			expected: "user-llm-text",
		},
		{
			name:     "send_text",
			constant: RTVIFrameTypeSendText,
			expected: "send-text",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

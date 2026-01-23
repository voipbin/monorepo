package transcribe

import (
	"testing"
)

func TestEventTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant string
		expected string
	}{
		{"event_type_transcribe_created", EventTypeTranscribeCreated, "transcribe_created"},
		{"event_type_transcribe_deleted", EventTypeTranscribeDeleted, "transcribe_deleted"},
		{"event_type_transcribe_progressing", EventTypeTranscribeProgressing, "transcribe_progressing"},
		{"event_type_transcribe_done", EventTypeTranscribeDone, "transcribe_done"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

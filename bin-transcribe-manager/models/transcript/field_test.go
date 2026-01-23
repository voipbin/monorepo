package transcript

import (
	"testing"
)

func TestFieldConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Field
		expected string
	}{
		{"field_id", FieldID, "id"},
		{"field_customer_id", FieldCustomerID, "customer_id"},
		{"field_transcribe_id", FieldTranscribeID, "transcribe_id"},
		{"field_direction", FieldDirection, "direction"},
		{"field_message", FieldMessage, "message"},
		{"field_tm_transcript", FieldTMTranscript, "tm_transcript"},
		{"field_tm_create", FieldTMCreate, "tm_create"},
		{"field_tm_delete", FieldTMDelete, "tm_delete"},
		{"field_deleted", FieldDeleted, "deleted"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

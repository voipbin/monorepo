package recording

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
		{"field_owner_type", FieldOwnerType, "owner_type"},
		{"field_owner_id", FieldOwnerID, "owner_id"},
		{"field_activeflow_id", FieldActiveflowID, "activeflow_id"},
		{"field_reference_type", FieldReferenceType, "reference_type"},
		{"field_reference_id", FieldReferenceID, "reference_id"},
		{"field_status", FieldStatus, "status"},
		{"field_format", FieldFormat, "format"},
		{"field_on_end_flow_id", FieldOnEndFlowID, "on_end_flow_id"},
		{"field_recording_name", FieldRecordingName, "recording_name"},
		{"field_filenames", FieldFilenames, "filenames"},
		{"field_asterisk_id", FieldAsteriskID, "asterisk_id"},
		{"field_channel_ids", FieldChannelIDs, "channel_ids"},
		{"field_tm_start", FieldTMStart, "tm_start"},
		{"field_tm_end", FieldTMEnd, "tm_end"},
		{"field_tm_create", FieldTMCreate, "tm_create"},
		{"field_tm_update", FieldTMUpdate, "tm_update"},
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

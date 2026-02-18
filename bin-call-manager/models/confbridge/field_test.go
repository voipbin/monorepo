package confbridge

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
		{"field_activeflow_id", FieldActiveflowID, "activeflow_id"},
		{"field_reference_type", FieldReferenceType, "reference_type"},
		{"field_reference_id", FieldReferenceID, "reference_id"},
		{"field_type", FieldType, "type"},
		{"field_status", FieldStatus, "status"},
		{"field_bridge_id", FieldBridgeID, "bridge_id"},
		{"field_flags", FieldFlags, "flags"},
		{"field_channel_call_ids", FieldChannelCallIDs, "channel_call_ids"},
		{"field_recording_id", FieldRecordingID, "recording_id"},
		{"field_recording_ids", FieldRecordingIDs, "recording_ids"},
		{"field_external_media_ids", FieldExternalMediaIDs, "external_media_ids"},
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

package conference

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
		{"field_confbridge_id", FieldConfbridgeID, "confbridge_id"},
		{"field_type", FieldType, "type"},
		{"field_status", FieldStatus, "status"},
		{"field_name", FieldName, "name"},
		{"field_detail", FieldDetail, "detail"},
		{"field_data", FieldData, "data"},
		{"field_timeout", FieldTimeout, "timeout"},
		{"field_pre_flow_id", FieldPreFlowID, "pre_flow_id"},
		{"field_post_flow_id", FieldPostFlowID, "post_flow_id"},
		{"field_conferencecall_ids", FieldConferencecallIDs, "conferencecall_ids"},
		{"field_recording_id", FieldRecordingID, "recording_id"},
		{"field_recording_ids", FieldRecordingIDs, "recording_ids"},
		{"field_transcribe_id", FieldTranscribeID, "transcribe_id"},
		{"field_transcribe_ids", FieldTranscribeIDs, "transcribe_ids"},
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

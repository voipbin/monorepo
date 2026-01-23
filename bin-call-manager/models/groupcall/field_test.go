package groupcall

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
		{"field_status", FieldStatus, "status"},
		{"field_flow_id", FieldFlowID, "flow_id"},
		{"field_source", FieldSource, "source"},
		{"field_destinations", FieldDestinations, "destinations"},
		{"field_master_call_id", FieldMasterCallID, "master_call_id"},
		{"field_master_groupcall_id", FieldMasterGroupcallID, "master_groupcall_id"},
		{"field_ring_method", FieldRingMethod, "ring_method"},
		{"field_answer_method", FieldAnswerMethod, "answer_method"},
		{"field_answer_call_id", FieldAnswerCallID, "answer_call_id"},
		{"field_call_ids", FieldCallIDs, "call_ids"},
		{"field_answer_groupcall_id", FieldAnswerGroupcallID, "answer_groupcall_id"},
		{"field_groupcall_ids", FieldGroupcallIDs, "groupcall_ids"},
		{"field_call_count", FieldCallCount, "call_count"},
		{"field_groupcall_count", FieldGroupcallCount, "groupcall_count"},
		{"field_dial_index", FieldDialIndex, "dial_index"},
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

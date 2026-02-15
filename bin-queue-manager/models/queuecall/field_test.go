package queuecall

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
		{"field_queue_id", FieldQueueID, "queue_id"},
		{"field_reference_type", FieldReferenceType, "reference_type"},
		{"field_reference_id", FieldReferenceID, "reference_id"},
		{"field_reference_activeflow_id", FieldReferenceActiveflowID, "reference_activeflow_id"},
		{"field_forward_action_id", FieldForwardActionID, "forward_action_id"},
		{"field_confbridge_id", FieldConfbridgeID, "confbridge_id"},
		{"field_source", FieldSource, "source"},
		{"field_routing_method", FieldRoutingMethod, "routing_method"},
		{"field_tag_ids", FieldTagIDs, "tag_ids"},
		{"field_status", FieldStatus, "status"},
		{"field_service_agent_id", FieldServiceAgentID, "service_agent_id"},
		{"field_timeout_wait", FieldTimeoutWait, "timeout_wait"},
		{"field_timeout_service", FieldTimeoutService, "timeout_service"},
		{"field_duration_waiting", FieldDurationWaiting, "duration_waiting"},
		{"field_duration_service", FieldDurationService, "duration_service"},
		{"field_tm_create", FieldTMCreate, "tm_create"},
		{"field_tm_service", FieldTMService, "tm_service"},
		{"field_tm_update", FieldTMUpdate, "tm_update"},
		{"field_tm_end", FieldTMEnd, "tm_end"},
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

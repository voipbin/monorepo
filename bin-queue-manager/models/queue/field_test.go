package queue

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
		{"field_name", FieldName, "name"},
		{"field_detail", FieldDetail, "detail"},
		{"field_routing_method", FieldRoutingMethod, "routing_method"},
		{"field_tag_ids", FieldTagIDs, "tag_ids"},
		{"field_execute", FieldExecute, "execute"},
		{"field_wait_flow_id", FieldWaitFlowID, "wait_flow_id"},
		{"field_wait_timeout", FieldWaitTimeout, "wait_timeout"},
		{"field_service_timeout", FieldServiceTimeout, "service_timeout"},
		{"field_wait_queuecall_ids", FieldWaitQueuecallIDs, "wait_queue_call_ids"},
		{"field_service_queuecall_ids", FieldServiceQueuecallIDs, "service_queue_call_ids"},
		{"field_total_incoming_count", FieldTotalIncomingCount, "total_incoming_count"},
		{"field_total_serviced_count", FieldTotalServicedCount, "total_serviced_count"},
		{"field_total_abandoned_count", FieldTotalAbandonedCount, "total_abandoned_count"},
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

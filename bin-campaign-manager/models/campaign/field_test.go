package campaign

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
		{"field_type", FieldType, "type"},
		{"field_execute", FieldExecute, "execute"},
		{"field_name", FieldName, "name"},
		{"field_detail", FieldDetail, "detail"},
		{"field_status", FieldStatus, "status"},
		{"field_service_level", FieldServiceLevel, "service_level"},
		{"field_end_handle", FieldEndHandle, "end_handle"},
		{"field_flow_id", FieldFlowID, "flow_id"},
		{"field_actions", FieldActions, "actions"},
		{"field_outplan_id", FieldOutplanID, "outplan_id"},
		{"field_outdial_id", FieldOutdialID, "outdial_id"},
		{"field_queue_id", FieldQueueID, "queue_id"},
		{"field_next_campaign_id", FieldNextCampaignID, "next_campaign_id"},
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

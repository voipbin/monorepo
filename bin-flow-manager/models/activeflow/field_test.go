package activeflow

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
		{"field_flow_id", FieldFlowID, "flow_id"},
		{"field_status", FieldStatus, "status"},
		{"field_reference_type", FieldReferenceType, "reference_type"},
		{"field_reference_id", FieldReferenceID, "reference_id"},
		{"field_reference_activeflow_id", FieldReferenceActiveflowID, "reference_activeflow_id"},
		{"field_on_complete_flow_id", FieldOnCompleteFlowID, "on_complete_flow_id"},
		{"field_stack_map", FieldStackMap, "stack_map"},
		{"field_current_stack_id", FieldCurrentStackID, "current_stack_id"},
		{"field_current_action", FieldCurrentAction, "current_action"},
		{"field_forward_stack_id", FieldForwardStackID, "forward_stack_id"},
		{"field_forward_action_id", FieldForwardActionID, "forward_action_id"},
		{"field_execute_count", FieldExecuteCount, "execute_count"},
		{"field_executed_actions", FieldExecutedActions, "executed_actions"},
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

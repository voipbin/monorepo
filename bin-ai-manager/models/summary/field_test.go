package summary

import "testing"

func TestFieldConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Field
		expected string
	}{
		{
			name:     "field_id",
			constant: FieldID,
			expected: "id",
		},
		{
			name:     "field_customer_id",
			constant: FieldCustomerID,
			expected: "customer_id",
		},
		{
			name:     "field_activeflow_id",
			constant: FieldActiveflowID,
			expected: "activeflow_id",
		},
		{
			name:     "field_on_end_flow_id",
			constant: FieldOnEndFlowID,
			expected: "on_end_flow_id",
		},
		{
			name:     "field_reference_type",
			constant: FieldReferenceType,
			expected: "reference_type",
		},
		{
			name:     "field_reference_id",
			constant: FieldReferenceID,
			expected: "reference_id",
		},
		{
			name:     "field_status",
			constant: FieldStatus,
			expected: "status",
		},
		{
			name:     "field_language",
			constant: FieldLanguage,
			expected: "language",
		},
		{
			name:     "field_content",
			constant: FieldContent,
			expected: "content",
		},
		{
			name:     "field_tm_create",
			constant: FieldTMCreate,
			expected: "tm_create",
		},
		{
			name:     "field_tm_update",
			constant: FieldTMUpdate,
			expected: "tm_update",
		},
		{
			name:     "field_tm_delete",
			constant: FieldTMDelete,
			expected: "tm_delete",
		},
		{
			name:     "field_deleted",
			constant: FieldDeleted,
			expected: "deleted",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

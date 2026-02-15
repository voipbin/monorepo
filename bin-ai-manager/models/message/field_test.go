package message

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
			name:     "field_aicall_id",
			constant: FieldAIcallID,
			expected: "aicall_id",
		},
		{
			name:     "field_direction",
			constant: FieldDirection,
			expected: "direction",
		},
		{
			name:     "field_role",
			constant: FieldRole,
			expected: "role",
		},
		{
			name:     "field_content",
			constant: FieldContent,
			expected: "content",
		},
		{
			name:     "field_tool_calls",
			constant: FieldToolCalls,
			expected: "tool_calls",
		},
		{
			name:     "field_tool_call_id",
			constant: FieldToolCallID,
			expected: "tool_call_id",
		},
		{
			name:     "field_tm_create",
			constant: FieldTMCreate,
			expected: "tm_create",
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

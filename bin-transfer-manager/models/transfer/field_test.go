package transfer

import (
	"testing"
)

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
			name:     "field_type",
			constant: FieldType,
			expected: "type",
		},
		{
			name:     "field_transferer_call_id",
			constant: FieldTransfererCallID,
			expected: "transferer_call_id",
		},
		{
			name:     "field_transferee_addresses",
			constant: FieldTransfereeAddresses,
			expected: "transferee_addresses",
		},
		{
			name:     "field_transferee_call_id",
			constant: FieldTransfereeCallID,
			expected: "transferee_call_id",
		},
		{
			name:     "field_groupcall_id",
			constant: FieldGroupcallID,
			expected: "groupcall_id",
		},
		{
			name:     "field_confbridge_id",
			constant: FieldConfbridgeID,
			expected: "confbridge_id",
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

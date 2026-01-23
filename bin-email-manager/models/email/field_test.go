package email

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
			name:     "field_activeflow_id",
			constant: FieldActiveflowID,
			expected: "activeflow_id",
		},
		{
			name:     "field_provider_type",
			constant: FieldProviderType,
			expected: "provider_type",
		},
		{
			name:     "field_provider_reference_id",
			constant: FieldProviderReferenceID,
			expected: "provider_reference_id",
		},
		{
			name:     "field_source",
			constant: FieldSource,
			expected: "source",
		},
		{
			name:     "field_destinations",
			constant: FieldDestinations,
			expected: "destinations",
		},
		{
			name:     "field_status",
			constant: FieldStatus,
			expected: "status",
		},
		{
			name:     "field_subject",
			constant: FieldSubject,
			expected: "subject",
		},
		{
			name:     "field_content",
			constant: FieldContent,
			expected: "content",
		},
		{
			name:     "field_attachments",
			constant: FieldAttachments,
			expected: "attachments",
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

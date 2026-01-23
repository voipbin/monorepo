package message

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
		{"field_source", FieldSource, "source"},
		{"field_targets", FieldTargets, "targets"},
		{"field_provider_name", FieldProviderName, "provider_name"},
		{"field_provider_reference_id", FieldProviderReferenceID, "provider_reference_id"},
		{"field_text", FieldText, "text"},
		{"field_medias", FieldMedias, "medias"},
		{"field_direction", FieldDirection, "direction"},
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

package provider

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
		{"field_type", FieldType, "type"},
		{"field_hostname", FieldHostname, "hostname"},
		{"field_tech_prefix", FieldTechPrefix, "tech_prefix"},
		{"field_tech_postfix", FieldTechPostfix, "tech_postfix"},
		{"field_tech_headers", FieldTechHeaders, "tech_headers"},
		{"field_name", FieldName, "name"},
		{"field_detail", FieldDetail, "detail"},
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

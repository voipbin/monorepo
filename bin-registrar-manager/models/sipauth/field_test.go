package sipauth

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
		{"field_reference_type", FieldReferenceType, "reference_type"},
		{"field_auth_types", FieldAuthTypes, "auth_types"},
		{"field_realm", FieldRealm, "realm"},
		{"field_username", FieldUsername, "username"},
		{"field_password", FieldPassword, "password"},
		{"field_allowed_ips", FieldAllowedIPs, "allowed_ips"},
		{"field_tm_create", FieldTMCreate, "tm_create"},
		{"field_tm_update", FieldTMUpdate, "tm_update"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

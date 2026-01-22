package participant

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
		{"field_chat_id", FieldChatID, "chat_id"},
		{"field_owner_type", FieldOwnerType, "owner_type"},
		{"field_owner_id", FieldOwnerID, "owner_id"},
		{"field_tm_joined", FieldTMJoined, "tm_joined"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

package message

import (
	"reflect"
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
		{"field_parent_id", FieldParentID, "parent_id"},
		{"field_owner_type", FieldOwnerType, "owner_type"},
		{"field_owner_id", FieldOwnerID, "owner_id"},
		{"field_type", FieldType, "type"},
		{"field_text", FieldText, "text"},
		{"field_medias", FieldMedias, "medias"},
		{"field_metadata", FieldMetadata, "metadata"},
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

func TestGetDBFields(t *testing.T) {
	fields := GetDBFields()

	expectedFields := []string{
		"id",
		"customer_id",
		"chat_id",
		"parent_id",
		"owner_type",
		"owner_id",
		"type",
		"text",
		"medias",
		"metadata",
		"tm_create",
		"tm_update",
		"tm_delete",
	}

	if !reflect.DeepEqual(fields, expectedFields) {
		t.Errorf("GetDBFields() = %v, expected %v", fields, expectedFields)
	}
}

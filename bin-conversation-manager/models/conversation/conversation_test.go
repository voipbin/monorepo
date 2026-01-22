package conversation

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestConversation(t *testing.T) {
	tests := []struct {
		name string

		accountID uuid.UUID
		convName  string
		detail    string
		convType  Type
		dialogID  string
	}{
		{
			name: "creates_conversation_with_all_fields",

			accountID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440001"),
			convName:  "Test Conversation",
			detail:    "A test conversation for unit testing",
			convType:  TypeMessage,
			dialogID:  "dialog-123",
		},
		{
			name: "creates_conversation_with_empty_fields",

			accountID: uuid.Nil,
			convName:  "",
			detail:    "",
			convType:  TypeNone,
			dialogID:  "",
		},
		{
			name: "creates_conversation_with_line_type",

			accountID: uuid.FromStringOrNil("550e8400-e29b-41d4-a716-446655440002"),
			convName:  "LINE Conversation",
			detail:    "A LINE messaging conversation",
			convType:  TypeLine,
			dialogID:  "line-chatroom-456",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Conversation{
				AccountID: tt.accountID,
				Name:      tt.convName,
				Detail:    tt.detail,
				Type:      tt.convType,
				DialogID:  tt.dialogID,
			}

			if c.AccountID != tt.accountID {
				t.Errorf("Wrong AccountID. expect: %s, got: %s", tt.accountID, c.AccountID)
			}
			if c.Name != tt.convName {
				t.Errorf("Wrong Name. expect: %s, got: %s", tt.convName, c.Name)
			}
			if c.Detail != tt.detail {
				t.Errorf("Wrong Detail. expect: %s, got: %s", tt.detail, c.Detail)
			}
			if c.Type != tt.convType {
				t.Errorf("Wrong Type. expect: %s, got: %s", tt.convType, c.Type)
			}
			if c.DialogID != tt.dialogID {
				t.Errorf("Wrong DialogID. expect: %s, got: %s", tt.dialogID, c.DialogID)
			}
		})
	}
}

func TestFieldConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Field
		expected string
	}{
		{
			name:     "field_deleted",
			constant: FieldDeleted,
			expected: "deleted",
		},
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
			name:     "field_owner_type",
			constant: FieldOwnerType,
			expected: "owner_type",
		},
		{
			name:     "field_owner_id",
			constant: FieldOwnerID,
			expected: "owner_id",
		},
		{
			name:     "field_account_id",
			constant: FieldAccountID,
			expected: "account_id",
		},
		{
			name:     "field_name",
			constant: FieldName,
			expected: "name",
		},
		{
			name:     "field_detail",
			constant: FieldDetail,
			expected: "detail",
		},
		{
			name:     "field_type",
			constant: FieldType,
			expected: "type",
		},
		{
			name:     "field_dialog_id",
			constant: FieldDialogID,
			expected: "dialog_id",
		},
		{
			name:     "field_self",
			constant: FieldSelf,
			expected: "self",
		},
		{
			name:     "field_peer",
			constant: FieldPeer,
			expected: "peer",
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{
			name:     "type_none",
			constant: TypeNone,
			expected: "",
		},
		{
			name:     "type_message",
			constant: TypeMessage,
			expected: "message",
		},
		{
			name:     "type_line",
			constant: TypeLine,
			expected: "line",
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

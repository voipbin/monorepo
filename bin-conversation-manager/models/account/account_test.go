package account

import (
	"testing"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/models/eventtopic"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Account carries an explicit subscription address on the global topic exchange
// (VOIP-1404/1419). The assertion pins the POINTER type: the event data reaches notifyhandler
// as a POINTER and the interface assertion matches the dynamic type.
var _ eventtopic.SubscriptionIdentifier = (*Account)(nil)

func Test_AccountEventSubscriptionID(t *testing.T) {
	accountID := uuid.Must(uuid.NewV4())
	customerID := uuid.Must(uuid.NewV4())
	messageFlowID := uuid.Must(uuid.NewV4())

	h := &Account{
		Identity: commonidentity.Identity{
			ID:         accountID,
			CustomerID: customerID,
		},
		Type:          TypeLine,
		MessageFlowID: messageFlowID,
	}

	res := h.EventSubscriptionID()
	if res != accountID.String() {
		t.Errorf("Wrong match. expect: %s, got: %s", accountID.String(), res)
	}
	if res == customerID.String() {
		t.Errorf("Account must not be addressed by its customer id. got: %s", res)
	}
	if res == messageFlowID.String() {
		t.Errorf("Account must not be addressed by its message flow id. got: %s", res)
	}
}

func TestAccount(t *testing.T) {
	tests := []struct {
		name string

		accType Type
		accName string
		detail  string
		secret  string
		token   string
	}{
		{
			name: "creates_account_with_all_fields",

			accType: TypeLine,
			accName: "LINE Account",
			detail:  "A LINE messaging account",
			secret:  "line-secret-123",
			token:   "line-token-456",
		},
		{
			name: "creates_account_with_empty_fields",

			accType: "",
			accName: "",
			detail:  "",
			secret:  "",
			token:   "",
		},
		{
			name: "creates_sms_account",

			accType: TypeSMS,
			accName: "SMS Account",
			detail:  "An SMS messaging account",
			secret:  "sms-secret-789",
			token:   "sms-token-012",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			a := &Account{
				Type:   tt.accType,
				Name:   tt.accName,
				Detail: tt.detail,
				Secret: tt.secret,
				Token:  tt.token,
			}

			if a.Type != tt.accType {
				t.Errorf("Wrong Type. expect: %s, got: %s", tt.accType, a.Type)
			}
			if a.Name != tt.accName {
				t.Errorf("Wrong Name. expect: %s, got: %s", tt.accName, a.Name)
			}
			if a.Detail != tt.detail {
				t.Errorf("Wrong Detail. expect: %s, got: %s", tt.detail, a.Detail)
			}
			if a.Secret != tt.secret {
				t.Errorf("Wrong Secret. expect: %s, got: %s", tt.secret, a.Secret)
			}
			if a.Token != tt.token {
				t.Errorf("Wrong Token. expect: %s, got: %s", tt.token, a.Token)
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
			name:     "field_secret",
			constant: FieldSecret,
			expected: "secret",
		},
		{
			name:     "field_token",
			constant: FieldToken,
			expected: "token",
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

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{
			name:     "type_line",
			constant: TypeLine,
			expected: "line",
		},
		{
			name:     "type_sms",
			constant: TypeSMS,
			expected: "sms",
		},
		{
			name:     "type_whatsapp",
			constant: TypeWhatsApp,
			expected: "whatsapp",
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

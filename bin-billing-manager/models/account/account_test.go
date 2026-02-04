package account

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestAccountStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())

	a := Account{
		Name:          "Test Account",
		Detail:        "Test account details",
		Type:          TypeNormal,
		Balance:       100.50,
		PaymentType:   PaymentTypePrepaid,
		PaymentMethod: PaymentMethodCreditCard,
		TMCreate:      "2024-01-01T00:00:00.000000Z",
		TMUpdate:      "2024-01-01T00:00:00.000000Z",
		TMDelete:      "9999-01-01T00:00:00.000000Z",
	}
	a.ID = id

	if a.ID != id {
		t.Errorf("Account.ID = %v, expected %v", a.ID, id)
	}
	if a.Name != "Test Account" {
		t.Errorf("Account.Name = %v, expected %v", a.Name, "Test Account")
	}
	if a.Type != TypeNormal {
		t.Errorf("Account.Type = %v, expected %v", a.Type, TypeNormal)
	}
	if a.Balance != 100.50 {
		t.Errorf("Account.Balance = %v, expected %v", a.Balance, 100.50)
	}
}

func TestTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Type
		expected string
	}{
		{"type_admin", TypeAdmin, "admin"},
		{"type_normal", TypeNormal, "normal"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestPaymentTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant PaymentType
		expected string
	}{
		{"payment_type_none", PaymentTypeNone, ""},
		{"payment_type_prepaid", PaymentTypePrepaid, "prepaid"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestPaymentMethodConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant PaymentMethod
		expected string
	}{
		{"payment_method_none", PaymentMethodNone, ""},
		{"payment_method_credit_card", PaymentMethodCreditCard, "credit card"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

package account

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func TestAccountStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	tmCreate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)

	a := Account{
		Name:          "Test Account",
		Detail:        "Test account details",
		Balance:       100.50,
		PaymentType:   PaymentTypePrepaid,
		PaymentMethod: PaymentMethodCreditCard,
		TMCreate:      &tmCreate,
		TMUpdate:      &tmUpdate,
		TMDelete:      nil,
	}
	a.ID = id

	if a.ID != id {
		t.Errorf("Account.ID = %v, expected %v", a.ID, id)
	}
	if a.Name != "Test Account" {
		t.Errorf("Account.Name = %v, expected %v", a.Name, "Test Account")
	}
	if a.Balance != 100.50 {
		t.Errorf("Account.Balance = %v, expected %v", a.Balance, 100.50)
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

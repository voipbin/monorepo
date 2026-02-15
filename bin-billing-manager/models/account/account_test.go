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
		BalanceCredit: 10050000,
		BalanceToken:  100,
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
	if a.BalanceCredit != 10050000 {
		t.Errorf("Account.BalanceCredit = %v, expected %v", a.BalanceCredit, 10050000)
	}
	if a.BalanceToken != 100 {
		t.Errorf("Account.BalanceToken = %v, expected %v", a.BalanceToken, 100)
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

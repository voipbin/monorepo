package billing

import (
	"testing"

	"github.com/gofrs/uuid"
)

func TestBillingStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	accountID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	b := Billing{
		AccountID:        accountID,
		Status:           StatusProgressing,
		ReferenceType:    ReferenceTypeCall,
		ReferenceID:      referenceID,
		CostPerUnit:      0.020,
		CostTotal:        0.40,
		BillingUnitCount: 20,
		TMBillingStart:   "2024-01-01T00:00:00.000000Z",
		TMBillingEnd:     "2024-01-01T00:20:00.000000Z",
		TMCreate:         "2024-01-01T00:00:00.000000Z",
		TMUpdate:         "2024-01-01T00:20:00.000000Z",
		TMDelete:         "9999-01-01T00:00:00.000000Z",
	}
	b.ID = id

	if b.ID != id {
		t.Errorf("Billing.ID = %v, expected %v", b.ID, id)
	}
	if b.AccountID != accountID {
		t.Errorf("Billing.AccountID = %v, expected %v", b.AccountID, accountID)
	}
	if b.Status != StatusProgressing {
		t.Errorf("Billing.Status = %v, expected %v", b.Status, StatusProgressing)
	}
	if b.ReferenceType != ReferenceTypeCall {
		t.Errorf("Billing.ReferenceType = %v, expected %v", b.ReferenceType, ReferenceTypeCall)
	}
}

func TestReferenceTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant ReferenceType
		expected string
	}{
		{"reference_type_none", ReferenceTypeNone, ""},
		{"reference_type_call", ReferenceTypeCall, "call"},
		{"reference_type_sms", ReferenceTypeSMS, "sms"},
		{"reference_type_number", ReferenceTypeNumber, "number"},
		{"reference_type_number_renew", ReferenceTypeNumberRenew, "number_renew"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestStatusConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant Status
		expected string
	}{
		{"status_progressing", StatusProgressing, "progressing"},
		{"status_end", StatusEnd, "end"},
		{"status_pending", StatusPending, "pending"},
		{"status_finished", StatusFinished, "finished"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if string(tt.constant) != tt.expected {
				t.Errorf("Wrong constant value. expect: %s, got: %s", tt.expected, tt.constant)
			}
		})
	}
}

func TestDefaultCostConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant float32
		expected float32
	}{
		{"default_cost_call", DefaultCostPerUnitReferenceTypeCall, 0.020},
		{"default_cost_sms", DefaultCostPerUnitReferenceTypeSMS, 0.008},
		{"default_cost_number", DefaultCostPerUnitReferenceTypeNumber, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %f, got: %f", tt.expected, tt.constant)
			}
		})
	}
}

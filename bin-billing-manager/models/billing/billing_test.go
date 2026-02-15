package billing

import (
	"testing"
	"time"

	"github.com/gofrs/uuid"
)

func TestBillingStruct(t *testing.T) {
	id := uuid.Must(uuid.NewV4())
	accountID := uuid.Must(uuid.NewV4())
	referenceID := uuid.Must(uuid.NewV4())

	tmBillingStart := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tmBillingEnd := time.Date(2024, 1, 1, 0, 20, 0, 0, time.UTC)
	tmCreate := time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
	tmUpdate := time.Date(2024, 1, 1, 0, 20, 0, 0, time.UTC)

	b := Billing{
		AccountID:         accountID,
		TransactionType:   TransactionTypeUsage,
		Status:            StatusProgressing,
		ReferenceType:     ReferenceTypeCall,
		ReferenceID:       referenceID,
		RateCreditPerUnit: 6000,
		AmountCredit:      -12000,
		BillableUnits:     2,
		UsageDuration:     65,
		TMBillingStart:    &tmBillingStart,
		TMBillingEnd:      &tmBillingEnd,
		TMCreate:          &tmCreate,
		TMUpdate:          &tmUpdate,
		TMDelete:          nil,
	}
	b.ID = id

	if b.ID != id {
		t.Errorf("Billing.ID = %v, expected %v", b.ID, id)
	}
	if b.AccountID != accountID {
		t.Errorf("Billing.AccountID = %v, expected %v", b.AccountID, accountID)
	}
	if b.TransactionType != TransactionTypeUsage {
		t.Errorf("Billing.TransactionType = %v, expected %v", b.TransactionType, TransactionTypeUsage)
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

func TestTransactionTypeConstants(t *testing.T) {
	tests := []struct {
		name     string
		constant TransactionType
		expected string
	}{
		{"transaction_type_usage", TransactionTypeUsage, "usage"},
		{"transaction_type_top_up", TransactionTypeTopUp, "top_up"},
		{"transaction_type_adjustment", TransactionTypeAdjustment, "adjustment"},
		{"transaction_type_refund", TransactionTypeRefund, "refund"},
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
		constant int64
		expected int64
	}{
		{"default_credit_call_pstn_outgoing", DefaultCreditPerUnitCallPSTNOutgoing, 6000},
		{"default_credit_sms", DefaultCreditPerUnitSMS, 8000},
		{"default_credit_number", DefaultCreditPerUnitNumber, 5000000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.constant != tt.expected {
				t.Errorf("Wrong constant value. expect: %d, got: %d", tt.expected, tt.constant)
			}
		})
	}
}

func TestCalculateBillableUnits(t *testing.T) {
	tests := []struct {
		name        string
		durationSec int
		expected    int
	}{
		{"zero seconds", 0, 0},
		{"negative seconds", -5, 0},
		{"large negative", -1000, 0},
		{"one second", 1, 1},
		{"thirty seconds", 30, 1},
		{"fifty-nine seconds", 59, 1},
		{"exactly one minute", 60, 1},
		{"sixty-one seconds", 61, 2},
		{"ninety seconds", 90, 2},
		{"exactly two minutes", 120, 2},
		{"two minutes one second", 121, 3},
		{"exactly five minutes", 300, 5},
		{"ten minutes one second", 601, 11},
		{"one hour", 3600, 60},
		{"one hour one second", 3601, 61},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := CalculateBillableUnits(tt.durationSec)
			if result != tt.expected {
				t.Errorf("CalculateBillableUnits(%d) = %d, expected %d", tt.durationSec, result, tt.expected)
			}
		})
	}
}

func TestGetCostInfo(t *testing.T) {
	tests := []struct {
		name              string
		costType          CostType
		expectedToken     int64
		expectedCredit    int64
	}{
		{"call_pstn_outgoing", CostTypeCallPSTNOutgoing, 0, DefaultCreditPerUnitCallPSTNOutgoing},
		{"call_pstn_incoming", CostTypeCallPSTNIncoming, 0, DefaultCreditPerUnitCallPSTNIncoming},
		{"call_vn", CostTypeCallVN, DefaultTokenPerUnitCallVN, DefaultCreditPerUnitCallVN},
		{"call_extension", CostTypeCallExtension, 0, 0},
		{"call_direct_ext", CostTypeCallDirectExt, 0, 0},
		{"sms", CostTypeSMS, DefaultTokenPerUnitSMS, DefaultCreditPerUnitSMS},
		{"number", CostTypeNumber, 0, DefaultCreditPerUnitNumber},
		{"number_renew", CostTypeNumberRenew, 0, DefaultCreditPerUnitNumber},
		{"none", CostTypeNone, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokenPerUnit, creditPerUnit := GetCostInfo(tt.costType)
			if tokenPerUnit != tt.expectedToken {
				t.Errorf("GetCostInfo(%s) tokenPerUnit = %d, expected %d", tt.costType, tokenPerUnit, tt.expectedToken)
			}
			if creditPerUnit != tt.expectedCredit {
				t.Errorf("GetCostInfo(%s) creditPerUnit = %d, expected %d", tt.costType, creditPerUnit, tt.expectedCredit)
			}
		})
	}
}

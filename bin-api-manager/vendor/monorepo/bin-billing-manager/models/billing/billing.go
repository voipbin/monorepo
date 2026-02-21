package billing

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// TransactionType defines the nature of the ledger entry
type TransactionType string

const (
	TransactionTypeUsage      TransactionType = "usage"
	TransactionTypeTopUp      TransactionType = "top_up"
	TransactionTypeAdjustment TransactionType = "adjustment"
	TransactionTypeRefund     TransactionType = "refund"
)

// Billing define
type Billing struct {
	commonidentity.Identity

	AccountID uuid.UUID `json:"account_id" db:"account_id,uuid"`

	// Transaction classification
	TransactionType TransactionType `json:"transaction_type" db:"transaction_type"`
	Status          Status          `json:"status" db:"status"`

	// Source context
	ReferenceType ReferenceType `json:"reference_type" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id" db:"reference_id,uuid"`
	CostType      CostType      `json:"cost_type" db:"cost_type"`

	// Usage measurement
	UsageDuration int `json:"usage_duration" db:"usage_duration"`
	BillableUnits int `json:"billable_units" db:"billable_units"`

	// Rates
	RateTokenPerUnit  int64 `json:"rate_token_per_unit" db:"rate_token_per_unit"`
	RateCreditPerUnit int64 `json:"rate_credit_per_unit" db:"rate_credit_per_unit"`

	// Ledger delta (signed: usage = negative, top_up/refund = positive)
	AmountToken  int64 `json:"amount_token" db:"amount_token"`
	AmountCredit int64 `json:"amount_credit" db:"amount_credit"`

	// Post-transaction balance snapshots
	BalanceTokenSnapshot  int64 `json:"balance_token_snapshot" db:"balance_token_snapshot"`
	BalanceCreditSnapshot int64 `json:"balance_credit_snapshot" db:"balance_credit_snapshot"`

	// Idempotency
	IdempotencyKey uuid.UUID `json:"idempotency_key" db:"idempotency_key,uuid"`

	// Billing timeframe
	TMBillingStart *time.Time `json:"tm_billing_start" db:"tm_billing_start"`
	TMBillingEnd   *time.Time `json:"tm_billing_end" db:"tm_billing_end"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// ReferenceType define
type ReferenceType string

// list of reference types
const (
	ReferenceTypeNone             ReferenceType = ""
	ReferenceTypeCall             ReferenceType = "call"
	ReferenceTypeCallExtension    ReferenceType = "call_extension"
	ReferenceTypeSMS              ReferenceType = "sms"
	ReferenceTypeNumber           ReferenceType = "number"
	ReferenceTypeNumberRenew      ReferenceType = "number_renew"
	ReferenceTypeCreditFreeTier    ReferenceType = "credit_free_tier"
	ReferenceTypeMonthlyAllowance  ReferenceType = "monthly_allowance"
	ReferenceTypeCreditAdjustment  ReferenceType = "credit_adjustment"
	ReferenceTypeTokenAdjustment   ReferenceType = "token_adjustment"
)

// Status define
type Status string

// list of status
const (
	StatusProgressing Status = "progressing"
	StatusEnd         Status = "end"
	StatusPending     Status = "pending"
	StatusFinished    Status = "finished"
)

// CalculateBillableUnits returns billable minutes (ceiling-rounded from seconds).
func CalculateBillableUnits(durationSec int) int {
	if durationSec <= 0 {
		return 0
	}
	return (durationSec + 59) / 60
}

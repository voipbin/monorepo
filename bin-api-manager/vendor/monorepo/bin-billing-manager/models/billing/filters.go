package billing

import (
	"time"

	"github.com/gofrs/uuid"
)

// FieldStruct defines allowed filters for Billing queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID              uuid.UUID       `filter:"id"`
	CustomerID      uuid.UUID       `filter:"customer_id"`
	AccountID       uuid.UUID       `filter:"account_id"`
	TransactionType TransactionType `filter:"transaction_type"`
	Status          Status          `filter:"status"`
	ReferenceType   ReferenceType   `filter:"reference_type"`
	ReferenceID     uuid.UUID       `filter:"reference_id"`
	CostType        CostType        `filter:"cost_type"`
	AmountCredit    int64           `filter:"amount_credit"`
	TMBillingStart  *time.Time      `filter:"tm_billing_start"`
	TMBillingEnd    *time.Time      `filter:"tm_billing_end"`
	Deleted         bool            `filter:"deleted"`
}

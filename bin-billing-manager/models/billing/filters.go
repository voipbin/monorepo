package billing

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Billing queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID             uuid.UUID     `filter:"id"`
	CustomerID     uuid.UUID     `filter:"customer_id"`
	AccountID      uuid.UUID     `filter:"account_id"`
	Status         Status        `filter:"status"`
	ReferenceType  ReferenceType `filter:"reference_type"`
	ReferenceID    uuid.UUID     `filter:"reference_id"`
	CostPerUnit    float64       `filter:"cost_per_unit"`
	CostTotal      float64       `filter:"cost_total"`
	TMBillingStart string        `filter:"tm_billing_start"`
	TMBillingEnd   string        `filter:"tm_billing_end"`
	Deleted        bool          `filter:"deleted"`
}

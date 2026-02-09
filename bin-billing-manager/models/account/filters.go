package account

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Account queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID            uuid.UUID     `filter:"id"`
	CustomerID    uuid.UUID     `filter:"customer_id"`
	Name          string        `filter:"name"`
	Type          Type          `filter:"type"`
	PlanType      PlanType      `filter:"plan_type"`
	Balance       float64       `filter:"balance"`
	PaymentType   PaymentType   `filter:"payment_type"`
	PaymentMethod PaymentMethod `filter:"payment_method"`
	Deleted       bool          `filter:"deleted"`
}

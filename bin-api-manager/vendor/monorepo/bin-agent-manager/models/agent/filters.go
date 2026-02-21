package agent

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Agent queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID         uuid.UUID  `filter:"id"`
	CustomerID uuid.UUID  `filter:"customer_id"`
	Username   string     `filter:"username"`
	Name       string     `filter:"name"`
	RingMethod RingMethod `filter:"ring_method"`
	Status     Status     `filter:"status"`
	Deleted    bool       `filter:"deleted"`
}

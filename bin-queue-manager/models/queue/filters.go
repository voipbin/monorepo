package queue

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Queue queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID            uuid.UUID     `filter:"id"`
	CustomerID    uuid.UUID     `filter:"customer_id"`
	Name          string        `filter:"name"`
	RoutingMethod RoutingMethod `filter:"routing_method"`
	Execute       Execute       `filter:"execute"`
	Deleted       bool          `filter:"deleted"`
}

package direct

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Direct queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	CustomerID   uuid.UUID `filter:"customer_id"`
	ResourceType string    `filter:"resource_type"`
	ResourceID   uuid.UUID `filter:"resource_id"`
	Hash         string    `filter:"hash"`
}

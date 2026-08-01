package schedule

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Schedule queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	CustomerID uuid.UUID `filter:"customer_id"`
	Enabled    bool      `filter:"enabled"`
	Deleted    bool      `filter:"deleted"`
}

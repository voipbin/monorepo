package tag

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Tag queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID         uuid.UUID `filter:"id"`
	CustomerID uuid.UUID `filter:"customer_id"`
	Name       string    `filter:"name"`
	Deleted    bool      `filter:"deleted"`
}

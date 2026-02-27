package team

import "github.com/gofrs/uuid"

// FieldStruct defines filterable fields for Team list queries.
type FieldStruct struct {
	CustomerID uuid.UUID `filter:"customer_id"`
	Name       string    `filter:"name"`
	Deleted    bool      `filter:"deleted"`
}

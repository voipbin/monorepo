package rag

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Rag queries.
// Used by utilhandler.ConvertFilters to validate and type-convert filter values.
type FieldStruct struct {
	CustomerID uuid.UUID `filter:"customer_id"`
	Deleted    bool      `filter:"deleted"`
}

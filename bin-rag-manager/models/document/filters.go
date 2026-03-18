package document

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Document queries.
// Used by utilhandler.ConvertFilters to validate and type-convert filter values.
type FieldStruct struct {
	CustomerID uuid.UUID `filter:"customer_id"`
	RagID      uuid.UUID `filter:"rag_id"`
	Status     Status    `filter:"status"`
	Deleted    bool      `filter:"deleted"`
}

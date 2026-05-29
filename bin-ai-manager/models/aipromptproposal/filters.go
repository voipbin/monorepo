package aipromptproposal

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for AIPromptProposal queries.
// Each field corresponds to a filterable database column.
type FieldStruct struct {
	CustomerID uuid.UUID `filter:"customer_id"`
	AIID       uuid.UUID `filter:"ai_id"`
	Status     Status    `filter:"status"`
	Deleted    bool      `filter:"deleted"`
}

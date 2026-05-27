package aiaudit

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for AIAudit queries.
// Each field corresponds to a filterable database column.
type FieldStruct struct {
	CustomerID uuid.UUID `filter:"customer_id"`
	AIcallID   uuid.UUID `filter:"aicall_id"`
	AIID       uuid.UUID `filter:"ai_id"`
	Status     Status    `filter:"status"`
	Deleted    bool      `filter:"deleted"`
}

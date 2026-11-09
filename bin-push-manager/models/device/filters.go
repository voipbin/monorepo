package device

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Device queries.
// The token is deliberately not a filter.
type FieldStruct struct {
	ID         uuid.UUID `filter:"id"`
	CustomerID uuid.UUID `filter:"customer_id"`
	AgentID    uuid.UUID `filter:"agent_id"`
	Platform   string    `filter:"platform"`
}

package mcpserver

import "github.com/gofrs/uuid"

// FieldStruct defines filterable fields for McpServer list queries.
type FieldStruct struct {
	CustomerID uuid.UUID `filter:"customer_id"`
	Name       string    `filter:"name"`
	Status     Status    `filter:"status"`
	Deleted    bool      `filter:"deleted"`
}

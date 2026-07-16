package session

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Session queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID         uuid.UUID `filter:"id"`
	CustomerID uuid.UUID `filter:"customer_id"`
	WidgetID   uuid.UUID `filter:"widget_id"`
	Status     Status    `filter:"status"`
	Deleted    bool      `filter:"deleted"`
}

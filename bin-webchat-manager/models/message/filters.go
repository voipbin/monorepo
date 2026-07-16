package message

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Message queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID         uuid.UUID `filter:"id"`
	CustomerID uuid.UUID `filter:"customer_id"`
	WidgetID   uuid.UUID `filter:"widget_id"`
	SessionID  uuid.UUID `filter:"session_id"`
	Direction  Direction `filter:"direction"`
	Status     Status    `filter:"status"`
	Deleted    bool      `filter:"deleted"`
}

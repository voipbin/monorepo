package message

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Message queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	CustomerID uuid.UUID `filter:"customer_id"`
	AIcallID   uuid.UUID `filter:"aicall_id"`
	Direction  Direction `filter:"direction"`
	Role       Role      `filter:"role"`
	Deleted    bool      `filter:"deleted"`
}

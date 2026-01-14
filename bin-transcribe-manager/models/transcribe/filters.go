package transcribe

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Transcribe queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID            uuid.UUID     `filter:"id"`
	CustomerID    uuid.UUID     `filter:"customer_id"`
	ReferenceType ReferenceType `filter:"reference_type"`
	ReferenceID   uuid.UUID     `filter:"reference_id"`
	Status        Status        `filter:"status"`
	Direction     Direction     `filter:"direction"`
	Deleted       bool          `filter:"deleted"`
}

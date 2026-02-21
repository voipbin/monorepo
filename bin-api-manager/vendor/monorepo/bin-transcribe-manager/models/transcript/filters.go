package transcript

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Transcript queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID            uuid.UUID `filter:"id"`
	CustomerID    uuid.UUID `filter:"customer_id"`
	TranscribeID  uuid.UUID `filter:"transcribe_id"`
	Direction     Direction `filter:"direction"`
	Deleted       bool      `filter:"deleted"`
}

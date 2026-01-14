package conferencecall

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Conferencecall queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID            uuid.UUID     `filter:"id"`
	CustomerID    uuid.UUID     `filter:"customer_id"`
	ActiveflowID  uuid.UUID     `filter:"activeflow_id"`
	ConferenceID  uuid.UUID     `filter:"conference_id"`
	ReferenceType ReferenceType `filter:"reference_type"`
	ReferenceID   uuid.UUID     `filter:"reference_id"`
	Status        Status        `filter:"status"`
	Deleted       bool          `filter:"deleted"`
}

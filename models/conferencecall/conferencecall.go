package conferencecall

import "github.com/gofrs/uuid"

// Conferencecall defines contents of conferencecall
type Conferencecall struct {
	ID           uuid.UUID `json:"id"`
	CustomerID   uuid.UUID `json:"customer_id"`
	ConferenceID uuid.UUID `json:"conference_id"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	Status Status `json:"status"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ReferenceType define
type ReferenceType string

// list of reference types
const (
	ReferenceTypeCall ReferenceType = "call"
)

// Status define
type Status string

// list of status
const (
	StatusJoining Status = "joining"
	StatusJoined  Status = "joined"
	StatusLeaving Status = "leaving"
	StatusLeaved  Status = "leaved"
)

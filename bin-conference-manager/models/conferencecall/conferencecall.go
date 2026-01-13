package conferencecall

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Conferencecall defines contents of conferencecall
type Conferencecall struct {
	commonidentity.Identity

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`
	ConferenceID uuid.UUID `json:"conference_id,omitempty" db:"conference_id,uuid"`

	ReferenceType ReferenceType `json:"reference_type,omitempty" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty" db:"reference_id,uuid"`

	Status Status `json:"status,omitempty" db:"status"`

	TMCreate string `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate string `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete string `json:"tm_delete,omitempty" db:"tm_delete"`
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

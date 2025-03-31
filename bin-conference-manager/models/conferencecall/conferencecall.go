package conferencecall

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Conferencecall defines contents of conferencecall
type Conferencecall struct {
	commonidentity.Identity

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
	ConferenceID uuid.UUID `json:"conference_id,omitempty"`

	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	Status Status `json:"status,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
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

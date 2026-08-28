package conferencecall

import (
	"time"

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

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404/1405). It is the parent ConferenceID, not the conferencecall's
// own ID: the conferencecall id is stable, but it is not the axis anybody subscribes on. A
// conferencecall id first becomes known to a subscriber inside its own joining event, so binding
// to it in advance is impossible, while every real consumption pattern (following one conference
// session's participants) is conference-scoped. Single-item retrieval stays available over RPC.
// Design §2.1/§2.3.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (h *Conferencecall) EventSubscriptionID() string {
	return h.ConferenceID.String()
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

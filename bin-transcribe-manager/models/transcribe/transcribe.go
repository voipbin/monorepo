package transcribe

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Transcribe struct
type Transcribe struct {
	commonidentity.Identity

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
	OnEndFlowID  uuid.UUID `json:"on_end_flow_id,omitempty"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"` // call/conference/recording's id

	Status    Status    `json:"status"`
	HostID    uuid.UUID `json:"host_id"`  // host id
	Language  string    `json:"language"` // BCP47 type's language code. en-US
	Direction Direction `json:"direction"`

	StreamingIDs []uuid.UUID `json:"streaming_ids"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ReferenceType define
type ReferenceType string

// list of ReferenceType defines
const (
	ReferenceTypeUnknown    ReferenceType = "unknown"
	ReferenceTypeRecording  ReferenceType = "recording"
	ReferenceTypeCall       ReferenceType = "call"
	ReferenceTypeConference ReferenceType = "conference"
	ReferenceTypeConfbridge ReferenceType = "confbridge"
)

// Direction define
type Direction string

// list of Direction defines
const (
	DirectionBoth Direction = "both"
	DirectionIn   Direction = "in"
	DirectionOut  Direction = "out"
)

// Status defines
type Status string

// list of statuses
const (
	// StatusInit        Status = "init"
	StatusProgressing Status = "progressing"
	StatusDone        Status = "done"
)

// IsUpdatableStatus returns true if the new status is updatable.
func IsUpdatableStatus(oldStatus, newStatus Status) bool {

	// |--------------------+-------------------+---------------+
	// | old \ new          | StatusProgressing	| StatusDone	|
	// |--------------------+-------------------+---------------+
	// | StatusProgressing  |         x         |       o       |
	// |--------------------+---------------+-------------------+
	// | StatusDone         |         x         |       x       |
	// |--------------------+---------------+-------------------+

	mapOldStatusProgressing := map[Status]bool{
		StatusProgressing: false,
		StatusDone:        true,
	}
	mapOldStatusDone := map[Status]bool{
		StatusProgressing: false,
		StatusDone:        false,
	}

	switch oldStatus {
	case StatusProgressing:
		return mapOldStatusProgressing[newStatus]
	case StatusDone:
		return mapOldStatusDone[newStatus]
	}

	return false
}

package transcribe

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Transcribe struct
type Transcribe struct {
	commonidentity.Identity

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`
	OnEndFlowID  uuid.UUID `json:"on_end_flow_id,omitempty" db:"on_end_flow_id,uuid"`

	ReferenceType ReferenceType `json:"reference_type" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id" db:"reference_id,uuid"` // call/conference/recording's id

	Status    Status    `json:"status" db:"status"`
	HostID    uuid.UUID `json:"host_id" db:"host_id,uuid"` // host id
	Language  string    `json:"language" db:"language"`    // BCP47 type's language code. en-US
	Direction Direction `json:"direction" db:"direction"`

	StreamingIDs []uuid.UUID `json:"streaming_ids" db:"streaming_ids,json"`

	// timestamp
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// ReferenceType define
type ReferenceType string

// list of ReferenceType defines
const (
	ReferenceTypeUnknown    ReferenceType = "unknown"
	ReferenceTypeRecording  ReferenceType = "recording"
	ReferenceTypeCall       ReferenceType = "call"
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

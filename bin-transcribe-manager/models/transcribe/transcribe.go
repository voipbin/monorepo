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
	Provider  Provider  `json:"provider" db:"provider"`

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

// Normalize returns the direction if it is a known value (both/in/out),
// otherwise it falls back to DirectionBoth. An empty or invalid direction
// (e.g. a typo) would otherwise flow into the Asterisk snoop layer and fail at
// runtime, so callers normalize the value before use. DirectionBoth is the
// safe catch-all default and matches the long-standing behavior where an
// omitted direction captured both legs, so this fallback is backward
// compatible. The match is case-sensitive (no trim or case-folding): the
// direction enum is exposed only in lowercase, so a mismatched case is treated
// as an invalid value rather than silently accepted.
func (d Direction) Normalize() Direction {
	switch d {
	case DirectionBoth, DirectionIn, DirectionOut:
		return d
	}
	return DirectionBoth
}

// Provider defines the STT provider type
type Provider string

// list of Provider defines
const (
	ProviderEmpty Provider = ""
	ProviderGCP   Provider = "gcp"
	ProviderAWS   Provider = "aws"
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

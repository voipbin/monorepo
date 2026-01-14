package recording

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Recording struct represent record information
type Recording struct {
	commonidentity.Identity
	commonidentity.Owner

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty" db:"activeflow_id,uuid"`
	ReferenceType ReferenceType `json:"reference_type,omitempty" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty" db:"reference_id,uuid"`
	Status        Status        `json:"status,omitempty" db:"status"`
	Format        Format        `json:"format,omitempty" db:"format"`

	OnEndFlowID uuid.UUID `json:"on_end_flow_id,omitempty" db:"on_end_flow_id,uuid"` // executed when recording ends

	RecordingName string   `json:"recording_name,omitempty" db:"recording_name"`
	Filenames     []string `json:"filenames,omitempty" db:"filenames,json"`

	AsteriskID string   `json:"asterisk_id,omitempty" db:"asterisk_id"`
	ChannelIDs []string `json:"channel_ids,omitempty" db:"channel_ids,json"` // snoop channel ids for recording

	TMStart string `json:"tm_start,omitempty" db:"tm_start"`
	TMEnd   string `json:"tm_end,omitempty" db:"tm_end"`

	TMCreate string `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate string `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete string `json:"tm_delete,omitempty" db:"tm_delete"`
}

// ReferenceType type
type ReferenceType string

// List of reference types
const (
	ReferenceTypeCall       ReferenceType = "call"       // call type.
	ReferenceTypeConfbridge ReferenceType = "confbridge" // confbridge type.
)

// Status type
type Status string

// List of record status
const (
	StatusInitiating Status = "initiating"
	StatusRecording  Status = "recording"
	StatusStopping   Status = "stopping"
	StatusEnded      Status = "ended"
)

// Format type
type Format string

// list of formats
const (
	FormatWAV Format = "wav"
)

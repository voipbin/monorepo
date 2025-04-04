package recording

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Recording struct represent record information
type Recording struct {
	commonidentity.Identity
	commonidentity.Owner

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty"`
	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`
	Status        Status        `json:"status,omitempty"`
	Format        Format        `json:"format,omitempty"`

	OnEndFlowID uuid.UUID `json:"on_end_flow_id,omitempty"` // executed when recording ends

	RecordingName string   `json:"recording_name,omitempty"`
	Filenames     []string `json:"filenames,omitempty"`

	AsteriskID string   `json:"asterisk_id,omitempty"`
	ChannelIDs []string `json:"channel_ids,omitempty"` // snoop channel ids for recording

	TMStart string `json:"tm_start,omitempty"`
	TMEnd   string `json:"tm_end,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
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

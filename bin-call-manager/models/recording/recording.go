package recording

import "github.com/gofrs/uuid"

// Recording struct represent record information
type Recording struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerType  OwnerType `json:"owner_type"`
	OwnerID    uuid.UUID `json:"owner_id"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`
	Status        Status        `json:"status"`
	Format        Format        `json:"format"`

	RecordingName string   `json:"recording_name"`
	Filenames     []string `json:"filenames"`

	AsteriskID string   `json:"asterisk_id"`
	ChannelIDs []string `json:"channel_ids"` // snoop channel ids for recording

	TMStart string `json:"tm_start"`
	TMEnd   string `json:"tm_end"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// OwnerType defines
type OwnerType string

// list of owner types
const (
	OwnerTypeNone  OwnerType = ""
	OwnerTypeAgent OwnerType = "agent" // the owner id is agent's id.
)

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

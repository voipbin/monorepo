package models

import "github.com/gofrs/uuid"

// Recording struct represent record information
type Recording struct {
	ID          uuid.UUID       `json:"id"`
	UserID      uint64          `json:"user_id"`
	Type        RecordingType   `json:"type"`
	ReferenceID uuid.UUID       `json:"reference_id"`
	Status      RecordingStatus `json:"status"`
	Format      string          `json:"format"`
	Filename    string          `json:"filename"`

	TMStart string `json:"tm_start"`
	TMEnd   string `json:"tm_end"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// RecordingType type
type RecordingType string

// List of types
const (
	RecordingTypeCall       RecordingType = "call"       // call type.
	RecordingTypeConference RecordingType = "conference" // conference type.
)

// RecordingStatus type
type RecordingStatus string

// List of record status
const (
	RecordingStatusInitiating RecordingStatus = "initiating"
	RecordingStatusRecording  RecordingStatus = "recording"
	RecordingStatusEnd        RecordingStatus = "ended"
)

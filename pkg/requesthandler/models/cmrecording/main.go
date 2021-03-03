package cmrecording

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models"
)

// Recording struct represent record information
type Recording struct {
	ID          uuid.UUID `json:"id"`
	UserID      uint64    `json:"user_id"`
	Type        Type      `json:"type"`
	ReferenceID uuid.UUID `json:"reference_id"`
	Status      Status    `json:"status"`
	Format      string    `json:"format"`
	Filename    string    `json:"filename"`

	AsteriskID string `json:"asterisk_id"`
	ChannelID  string `json:"channel_id"`

	TMStart string `json:"tm_start"`
	TMEnd   string `json:"tm_end"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type type
type Type string

// List of types
const (
	TypeCall       Type = "call"       // call type.
	TypeConference Type = "conference" // conference type.
)

// Status type
type Status string

// List of record status
const (
	StatusInitiating Status = "initiating"
	StatusRecording  Status = "recording"
	StatusEnd        Status = "ended"
)

// Convert returns conference.Conference from cmconference.Conference
func (h *Recording) Convert() *models.Recording {
	res := &models.Recording{
		ID:     h.ID,
		UserID: h.UserID,
		Type:   models.RecordingType(h.Type),

		ReferenceID: h.ReferenceID,
		Status:      models.RecordingStatus(h.Status),
		Format:      h.Format,
		Filename:    h.Filename,

		TMStart: h.TMStart,
		TMEnd:   h.TMEnd,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}

	return res
}

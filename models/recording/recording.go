package recording

import (
	"github.com/gofrs/uuid"
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

// Recording struct represent record information
type Recording struct {
	ID          uuid.UUID `json:"id"`
	CustomerID  uuid.UUID `json:"customer_id"`
	Type        Type      `json:"type"`
	ReferenceID uuid.UUID `json:"reference_id"`
	Status      Status    `json:"status"`
	Format      string    `json:"format"`
	Filename    string    `json:"filename"`

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
func Convert(h *cmrecording.Recording) *Recording {
	res := &Recording{
		ID:         h.ID,
		CustomerID: h.CustomerID,
		Type:       Type(h.Type),

		ReferenceID: h.ReferenceID,
		Status:      Status(h.Status),
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

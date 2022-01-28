package recording

import "github.com/gofrs/uuid"

// Recording struct represent record information
type Recording struct {
	ID          uuid.UUID `json:"id"`
	CustomerID  uuid.UUID `json:"customer_id"`
	Type        Type      `json:"type"`
	ReferenceID uuid.UUID `json:"reference_id"`
	Status      Status    `json:"status"`
	Format      string    `json:"format"`
	Filename    string    `json:"filename"`
	WebhookURI  string    `json:"webhook_uri"`

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

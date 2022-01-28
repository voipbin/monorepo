package transcribe

import "github.com/gofrs/uuid"

// Transcribe struct
type Transcribe struct {
	ID            uuid.UUID    `json:"id"`             // Transcribe id
	CustomerID    uuid.UUID    `json:"customer_id"`    // customer
	Type          Type         `json:"type"`           // type
	ReferenceID   uuid.UUID    `json:"reference_id"`   // call/conference/recording's id
	HostID        uuid.UUID    `json:"host_id"`        // host id
	Language      string       `json:"language"`       // BCP47 type's language code. en-US
	WebhookURI    string       `json:"webhook_uri"`    // webhook destination uri
	WebhookMethod string       `json:"webhook_method"` // webhook method
	Transcripts   []Transcript `json:"transcripts"`    // transcripts

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type define
type Type string

// list of Types
const (
	TypeUnknown    Type = "unknown"
	TypeRecording  Type = "recording"
	TypeCall       Type = "call"
	TypeConference Type = "conference"
)

// TranscriptDirection define
type TranscriptDirection string

// list of TranscriptDirections
const (
	TranscriptDirectionBoth TranscriptDirection = "both"
	TranscriptDirectionIn   TranscriptDirection = "in"
	TranscriptDirectionOut  TranscriptDirection = "out"
)

// Transcript struct
type Transcript struct {
	Direction TranscriptDirection `json:"direction"` // direction. in/out
	Message   string              `json:"message"`   // message
	TMCreate  string              `json:"tm_create"` // timestamp
}

// TranscriptWebhook defines webhook message for transcript
type TranscriptWebhook struct {
	ID          uuid.UUID `json:"id"`
	Type        Type      `json:"type"`
	ReferenceID uuid.UUID `json:"reference_id"`
	Transcript
}

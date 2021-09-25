package transcribe

import "github.com/gofrs/uuid"

// Transcribe struct
type Transcribe struct {
	ID            uuid.UUID    `json:"id"`             // Transcribe id
	Type          Type         `json:"type"`           // type
	ReferenceID   uuid.UUID    `json:"reference_id"`   // recording's id
	Language      string       `json:"language"`       // BCP47 type's language code. en-US
	WebhookURI    string       `json:"webhook_uri"`    // webhook destination uri
	WebhookMethod string       `json:"webhook_method"` // webhook method
	Transcripts   []Transcript `json:"transcripts"`    // transcripts
}

// Type define
type Type string

// list of Types
const (
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

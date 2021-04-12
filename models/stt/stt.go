package stt

import "github.com/gofrs/uuid"

// STT struct
type STT struct {
	ID            uuid.UUID `json:"id"`             // STT id
	Type          Type      `json:"type"`           // type
	ReferenceID   uuid.UUID `json:"reference_id"`   // recording's id
	Language      string    `json:"language"`       // BCP47 type's language code. en-US
	WebhookURI    string    `json:"webhook_uri"`    // webhook destination uri
	WebhookMethod string    `json:"webhook_method"` // webhook method
	Transcript    string    `json:"transcript"`     // transcript
}

// Type define
type Type string

// list of Types
const (
	TypeRecording  Type = "recording"
	TypeCall       Type = "call"
	TypeConference Type = "conference"
)

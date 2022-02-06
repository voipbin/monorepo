package transcribe

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/common"
	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcript"
)

// Transcribe struct
type Transcribe struct {
	ID          uuid.UUID `json:"id"`           // Transcribe id
	CustomerID  uuid.UUID `json:"customer_id"`  // customer
	Type        Type      `json:"type"`         // type
	ReferenceID uuid.UUID `json:"reference_id"` // call/conference/recording's id

	HostID    uuid.UUID        `json:"host_id"`  // host id
	Language  string           `json:"language"` // BCP47 type's language code. en-US
	Direction common.Direction `json:"direction"`

	Transcripts []transcript.Transcript `json:"transcripts"` // transcripts

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

package transcript

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Transcript struct
type Transcript struct {
	commonidentity.Identity

	TranscribeID uuid.UUID `json:"transcribe_id" db:"transcribe_id,uuid"`

	Direction Direction `json:"direction" db:"direction"` // direction. in/out
	Message   string    `json:"message" db:"message"`     // message

	TMTranscript *time.Time `json:"tm_transcript" db:"tm_transcript"` // timestamp transcripted. 0001-01-01 00:00:00.00000 points begining of the transcribe craete time.

	TMCreate *time.Time `json:"tm_create" db:"tm_create"` // timestamp create
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"` // timestamp delete
}

// EventSubscriptionID returns the subscription address of the global topic exchange
// `bin-manager.event` (VOIP-1404). It is the parent TranscribeID, not the transcript's own ID:
// every final result gets a new transcript-id, so the own ID is not an address anybody could
// bind to in advance. Subscribers follow one transcription session.
//
// The receiver is a pointer because the event data reaches notifyhandler as a pointer and the
// eventtopic.SubscriptionIdentifier assertion matches the dynamic type.
func (h *Transcript) EventSubscriptionID() string {
	return h.TranscribeID.String()
}

// Direction define
type Direction string

// list of Direction
const (
	DirectionBoth Direction = "both"
	DirectionIn   Direction = "in"
	DirectionOut  Direction = "out"
)

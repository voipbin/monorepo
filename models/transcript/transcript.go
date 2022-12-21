package transcript

import (
	"github.com/gofrs/uuid"
)

// Transcript struct
type Transcript struct {
	ID           uuid.UUID `json:"id"`
	CustomerID   uuid.UUID `json:"customer_id"`
	TranscribeID uuid.UUID `json:"transcribe_id"`

	Direction Direction `json:"direction"` // direction. in/out
	Message   string    `json:"message"`   // message

	TMTranscript string `json:"tm_transcript"` // timestamp transcripted. 0000-00-00 00:00:00.00000 points begining of the transcribe craete time.

	TMCreate string `json:"tm_create"` // timestamp create
	TMDelete string `json:"tm_delete"` // timestamp delete
}

// Direction define
type Direction string

// list of Direction
const (
	DirectionBoth Direction = "both"
	DirectionIn   Direction = "in"
	DirectionOut  Direction = "out"
)

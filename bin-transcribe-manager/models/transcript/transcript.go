package transcript

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Transcript struct
type Transcript struct {
	commonidentity.Identity

	TranscribeID uuid.UUID `json:"transcribe_id" db:"transcribe_id,uuid"`

	Direction Direction `json:"direction" db:"direction"` // direction. in/out
	Message   string    `json:"message" db:"message"`     // message

	TMTranscript string `json:"tm_transcript" db:"tm_transcript"` // timestamp transcripted. 0001-01-01 00:00:00.00000 points begining of the transcribe craete time.

	TMCreate string `json:"tm_create" db:"tm_create"` // timestamp create
	TMDelete string `json:"tm_delete" db:"tm_delete"` // timestamp delete
}

// Direction define
type Direction string

// list of Direction
const (
	DirectionBoth Direction = "both"
	DirectionIn   Direction = "in"
	DirectionOut  Direction = "out"
)

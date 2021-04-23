package request

import (
	"github.com/gofrs/uuid"
)

// BodyTranscribesPOST defines request body for /v1.0/transcribes POST
type BodyTranscribesPOST struct {
	RecordingID uuid.UUID `json:"recording_id"`
	Language    string    `json:"language"`
}

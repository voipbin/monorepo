package streaming

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-transcribe-manager/models/transcript"
)

// Streaming defines current streaming detail
type Streaming struct {
	ID           uuid.UUID            `json:"id"`
	CustomerID   uuid.UUID            `json:"customer_id"`
	TranscribeID uuid.UUID            `json:"transcribe_id"`
	Language     string               `json:"language"`
	Direction    transcript.Direction `json:"direction"`
}

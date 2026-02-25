package streaming

import (
	"time"

	"github.com/gofrs/uuid"
	"github.com/gorilla/websocket"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-transcribe-manager/models/transcript"
)

// Streaming defines current streaming detail
type Streaming struct {
	commonidentity.Identity

	TranscribeID uuid.UUID            `json:"transcribe_id"`
	Language     string               `json:"language"`
	Direction    transcript.Direction `json:"direction"`

	ConnAst *websocket.Conn `json:"-"` // WebSocket connection to Asterisk
}

// NewSpeech creates a Speech event from the streaming session and per-event data.
func (h *Streaming) NewSpeech(message string, tmEvent *time.Time) *Speech {
	return &Speech{
		Identity: commonidentity.Identity{
			ID:         uuid.Must(uuid.NewV4()),
			CustomerID: h.CustomerID,
		},
		StreamingID:  h.ID,
		TranscribeID: h.TranscribeID,
		Language:     h.Language,
		Direction:    h.Direction,
		Message:      message,
		TMEvent:      tmEvent,
		TMCreate:     tmEvent,
	}
}

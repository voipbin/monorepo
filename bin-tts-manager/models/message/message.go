package message

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type Message struct {
	commonidentity.Identity

	StreamingID uuid.UUID `json:"streaming_id,omitempty"` // Current streaming session

	TotalMessage  string `json:"total_message,omitempty"`  // Total message
	PlayedMessage string `json:"played_message,omitempty"` // Played message to be synthesized

	TotalCount  int `json:"total_count,omitempty"` // Total number of times the message should be played
	PlayedCount int `json:"count,omitempty"`       // Number of times the message has been played

	Finish bool `json:"message_finish,omitempty"` // Whether the message has finished playing
}

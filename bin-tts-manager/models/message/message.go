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

	Finish bool `json:"message_finish,omitempty"` // Whether the message has finished playing
}

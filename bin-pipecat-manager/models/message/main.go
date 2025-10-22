package message

import (
	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type Message struct {
	identity.Identity

	PipecatCallID uuid.UUID `json:"pipecat_call_id,omitempty"`

	Text string `json:"text,omitempty"`
}

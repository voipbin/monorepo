package dtmf

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

type DTMF struct {
	commonidentity.Identity

	CallID   uuid.UUID `json:"call_id,omitempty"`
	Digit    string    `json:"digit,omitempty"`
	Duration int       `json:"duration,omitempty"` // in milliseconds

	TMCreate string `json:"tm_create,omitempty"`
}

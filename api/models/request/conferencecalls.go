package request

import (
	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
)

// BodyConferencecallsPOST is rquest body define for POST /conferencecalls
type BodyConferencecallsPOST struct {
	ConferenceID  uuid.UUID                    `json:"conference_id"`
	ReferenceType conferencecall.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID                    `json:"reference_id"`
}

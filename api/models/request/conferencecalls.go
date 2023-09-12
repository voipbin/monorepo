package request

import (
	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
)

// ParamConferencecallsGET is rquest param define for
// GET /v1.0/conferencecalls
type ParamConferencecallsGET struct {
	Pagination
}

// BodyConferencecallsPOST is rquest body define for
// POST /v1.0/conferencecalls
type BodyConferencecallsPOST struct {
	ConferenceID  uuid.UUID                    `json:"conference_id"`
	ReferenceType conferencecall.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID                    `json:"reference_id"`
}

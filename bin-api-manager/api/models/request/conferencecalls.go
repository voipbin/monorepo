package request

import (
	"monorepo/bin-conference-manager/models/conferencecall"

	"github.com/gofrs/uuid"
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

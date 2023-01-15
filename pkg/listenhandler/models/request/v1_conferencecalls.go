package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
)

// V1DataConferencecallsPost is
// v1 data type request struct for
// /v1/conferencecalls" PUT
type V1DataConferencecallsPost struct {
	ConferenceID  uuid.UUID                    `json:"conference_id"`
	ReferenceType conferencecall.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID                    `json:"reference_id"`
}

// V1DataConferencecallsIDHealthCheckPost is
// v1 data type request struct for
// /v1/conferencecalls/<conferencecall-id>/health-check POST
type V1DataConferencecallsIDHealthCheckPost struct {
	RetryCount int `json:"retry_count"`
}

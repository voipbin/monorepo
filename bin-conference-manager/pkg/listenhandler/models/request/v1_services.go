package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
)

// V1DataServicesTypeConferencecallPost is
// v1 data type request struct for
// /v1/services/conferencecall" POST
type V1DataServicesTypeConferencecallPost struct {
	ConferenceID  uuid.UUID                    `json:"conference_id"`
	ReferenceType conferencecall.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID                    `json:"reference_id"`
}

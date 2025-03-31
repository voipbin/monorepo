package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-conference-manager/models/conferencecall"
)

// V1DataServicesTypeConferencecallPost is
// v1 data type request struct for
// /v1/services/conferencecall" POST
type V1DataServicesTypeConferencecallPost struct {
	ActiveflowID  uuid.UUID                    `json:"activeflow_id,omitempty"`
	ConferenceID  uuid.UUID                    `json:"conference_id,omitempty"`
	ReferenceType conferencecall.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID                    `json:"reference_id,omitempty"`
}

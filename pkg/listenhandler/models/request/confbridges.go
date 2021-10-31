package request

import (
	"github.com/gofrs/uuid"
)

// V1DataConfbridgesPost is
// v1 data type request struct for
// /v1/confbridges/<id>" POST
type V1DataConfbridgesPost struct {
	ConferenceID uuid.UUID `json:"conference_id"`
}

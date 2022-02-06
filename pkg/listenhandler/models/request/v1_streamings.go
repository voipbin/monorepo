package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// V1DataStreamingsPost is
// v1 data type request struct for
// /v1/streamings POST
type V1DataStreamingsPost struct {
	CustomerID    uuid.UUID       `json:"customer_id"`    // customer id
	ReferenceID   uuid.UUID       `json:"reference_id"`   // call/conference id
	ReferenceType transcribe.Type `json:"reference_type"` // reference type. call/conference
	Language      string          `json:"language"`       // BCP47 type's language code. en-US
}

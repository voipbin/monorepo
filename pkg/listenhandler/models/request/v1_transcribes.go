package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// V1DataTranscribesPost is
// v1 data type request struct for
// /v1/transcribes POST
type V1DataTranscribesPost struct {
	CustomerID    uuid.UUID                `json:"customer_id"`    // customer id
	ReferenceType transcribe.ReferenceType `json:"reference_type"` // reference type. call/conference/recording, ...
	ReferenceID   uuid.UUID                `json:"reference_id"`   // reference id
	Language      string                   `json:"language"`       // BCP47 type's language code. en-US
	Direction     transcribe.Direction     `json:"direction"`
}

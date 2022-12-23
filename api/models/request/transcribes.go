package request

import (
	"github.com/gofrs/uuid"
	tmtranscribe "gitlab.com/voipbin/bin-manager/transcribe-manager.git/models/transcribe"
)

// BodyTranscribesPOST defines request body for /v1.0/transcribes POST
type BodyTranscribesPOST struct {
	ReferenceType tmtranscribe.ReferenceType `json:"transcribe_type"`
	ReferenceID   uuid.UUID                  `json:"reference_id"`
	Language      string                     `json:"language"`
	Direction     tmtranscribe.Direction     `json:"direction"`
}

// ParamTranscribesGET is rquest param define for GET /transcribes
type ParamTranscribesGET struct {
	Pagination
}

package request

import (
	tmtranscribe "monorepo/bin-transcribe-manager/models/transcribe"

	"github.com/gofrs/uuid"
)

// BodyTranscribesPOST defines request body for
// POST /v1.0/transcribes
type BodyTranscribesPOST struct {
	ReferenceType TranscribeReferenceType `json:"transcribe_type"`
	ReferenceID   uuid.UUID               `json:"reference_id"`
	Language      string                  `json:"language"`
	Direction     tmtranscribe.Direction  `json:"direction"`
}

// TranscribeReferenceType define
type TranscribeReferenceType string

// list of TranscribeReferenceType types
const (
	TranscribeReferenceTypeCall       TranscribeReferenceType = "call"       // call
	TranscribeReferenceTypeConference TranscribeReferenceType = "conference" // conference
	TranscribeReferenceTypeRecording  TranscribeReferenceType = "recording"  // recording
)

// ParamTranscribesGET is rquest param define for
// GET /v1.0/transcribes
type ParamTranscribesGET struct {
	Pagination
}

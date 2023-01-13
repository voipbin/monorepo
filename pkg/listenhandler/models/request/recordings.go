package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

// V1DataRecordingsGET is rquest param define for GET /recordings
type V1DataRecordingsGET struct {
	UserID uint64 `json:"user_id"`
	Pagination
}

// V1DataRecordingsPost is
// v1 data type request struct for
// /v1/recordings POST
type V1DataRecordingsPost struct {
	ReferenceType recording.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID               `json:"reference_id"`
	Format        recording.Format        `json:"format"`         // default wav
	EndOfSilence  int                     `json:"end_of_silence"` // milliseconds
	EndOfKey      string                  `json:"end_of_key"`
	Duration      int                     `json:"duration"` // milliseconds
}

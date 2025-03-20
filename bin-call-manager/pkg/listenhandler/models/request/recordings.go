package request

import (
	"monorepo/bin-call-manager/models/recording"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// V1DataRecordingsGET is rquest param define for GET /recordings
type V1DataRecordingsGET struct {
	UserID uint64 `json:"user_id,omitempty"`
	Pagination
}

// V1DataRecordingsPost is
// v1 data type request struct for
// /v1/recordings POST
type V1DataRecordingsPost struct {
	OwnerType     commonidentity.OwnerType `json:"owner_type,omitempty"`
	OwnerID       uuid.UUID                `json:"owner_id,omitempty"`
	ReferenceType recording.ReferenceType  `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID                `json:"reference_id,omitempty"`
	Format        recording.Format         `json:"format,omitempty"`         // default wav
	EndOfSilence  int                      `json:"end_of_silence,omitempty"` // milliseconds
	EndOfKey      string                   `json:"end_of_key,omitempty"`
	Duration      int                      `json:"duration,omitempty"` // milliseconds
	OnEndFlowID   uuid.UUID                `json:"on_end_flow_id,omitempty"`
}

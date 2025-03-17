package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aicall"
)

// V1DataServicesTypeAIcallPost is
// v1 data type request struct for
// /v1/services/aicall POST
type V1DataServicesTypeAIcallPost struct {
	AIID uuid.UUID `json:"ai_id"`

	ActiveflowID  uuid.UUID            `json:"activeflow_id"`
	ReferenceType aicall.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID            `json:"reference_id"`

	Gender   aicall.Gender `json:"gender"`
	Language string        `json:"language"`
}

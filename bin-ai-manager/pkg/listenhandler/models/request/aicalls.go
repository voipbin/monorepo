package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aicall"
)

// V1DataAIcallsPost is
// v1 data type request struct for
// /v1/aicalls POST
type V1DataAIcallsPost struct {
	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`

	AIID uuid.UUID `json:"ai_id,omitempty"`

	ReferenceType aicall.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID            `json:"reference_id,omitempty"`

	Gender   aicall.Gender `json:"gender,omitempty"`
	Language string        `json:"language,omitempty"`
}

// V1DataAIcallsIDMessagesPost is
// v1 data type request struct for
// /v1/aicalls/<ai-id>/messages POST
type V1DataAIcallsIDMessagesPost struct {
	Role aicall.MessageRole `json:"role,omitempty"`
	Text string             `json:"text,omitempty"`
}

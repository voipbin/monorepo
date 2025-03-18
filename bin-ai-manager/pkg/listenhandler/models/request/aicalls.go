package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/aicall"
)

// V1DataAIcallsPost is
// v1 data type request struct for
// /v1/aicalls POST
type V1DataAIcallsPost struct {
	AIID uuid.UUID `json:"ai_id"`

	ReferenceType aicall.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID            `json:"reference_id"`

	Gender   aicall.Gender `json:"gender"`
	Language string        `json:"language"`
}

// V1DataAIcallsIDMessagesPost is
// v1 data type request struct for
// /v1/aicalls/<ai-id>/messages POST
type V1DataAIcallsIDMessagesPost struct {
	Role aicall.MessageRole `json:"role,omitempty"`
	Text string             `json:"text,omitempty"`
}

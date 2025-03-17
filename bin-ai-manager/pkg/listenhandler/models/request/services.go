package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/chatbotcall"
)

// V1DataServicesTypeChatbotcallPost is
// v1 data type request struct for
// /v1/services/chatbotcall POST
type V1DataServicesTypeChatbotcallPost struct {
	ChatbotID uuid.UUID `json:"chatbot_id"`

	ActiveflowID  uuid.UUID                 `json:"activeflow_id"`
	ReferenceType chatbotcall.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID                 `json:"reference_id"`

	Gender   chatbotcall.Gender `json:"gender"`
	Language string             `json:"language"`
}

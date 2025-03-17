package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/chatbotcall"
)

// V1DataChatbotcallsPost is
// v1 data type request struct for
// /v1/chatbotcalls POST
type V1DataChatbotcallsPost struct {
	ChatbotID uuid.UUID `json:"chatbot_id"`

	ReferenceType chatbotcall.ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID                 `json:"reference_id"`

	Gender   chatbotcall.Gender `json:"gender"`
	Language string             `json:"language"`
}

// V1DataChatbotcallsIDMessagesPost is
// v1 data type request struct for
// /v1/chatbotcalls/<chatbotcall-id>/messages POST
type V1DataChatbotcallsIDMessagesPost struct {
	Role chatbotcall.MessageRole `json:"role,omitempty"`
	Text string                  `json:"text,omitempty"`
}

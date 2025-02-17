package chatbotcall

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-common-handler/models/identity"
)

// Chatbotcall define
type Chatbotcall struct {
	identity.Identity

	ChatbotID          uuid.UUID           `json:"chatbot_id,omitempty"`
	ChatbotEngineType  chatbot.EngineType  `json:"chatbot_engine_type,omitempty"`
	ChatbotEngineModel chatbot.EngineModel `json:"chatbot_engine_model,omitempty"`

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty"`
	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	ConfbridgeID uuid.UUID `json:"confbridge_id,omitempty"`
	TranscribeID uuid.UUID `json:"transcribe_id,omitempty"`

	Status Status `json:"status,omitempty"`

	Gender   Gender `json:"gender,omitempty"`
	Language string `json:"language,omitempty"`

	Messages []Message `json:"messages,omitempty"`

	TMEnd    string `json:"tm_end,omitempty"`
	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// ReferenceType define
type ReferenceType string

// list of reference types
const (
	ReferenceTypeCall ReferenceType = "call"
)

// Status define
type Status string

// list of Statuses
const (
	StatusInitiating  Status = "initiating"
	StatusProgressing Status = "progressing"
	StatusEnd         Status = "end"
)

// Gender define
type Gender string

// list of genders
const (
	GenderMale    Gender = "male"
	GenderFemale  Gender = "female"
	GenderNuetral Gender = "neutral"
)

// Message defines
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

package chatbotcall

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
)

// Chatbotcall define
type Chatbotcall struct {
	ID                uuid.UUID          `json:"id"`
	CustomerID        uuid.UUID          `json:"customer_id"`
	ChatbotID         uuid.UUID          `json:"chatbot_id"`
	ChatbotEngineType chatbot.EngineType `json:"chatbot_engine_type"`

	ActiveflowID  uuid.UUID     `json:"activeflow_id"`
	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	ConfbridgeID uuid.UUID `json:"confbridge_id"`
	TranscribeID uuid.UUID `json:"transcribe_id"`

	Status Status `json:"status"`

	Gender   Gender `json:"gender"`
	Language string `json:"language"`

	Messages []Message `json:"messages"`

	TMEnd    string `json:"tm_end"`
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
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

package chatbotcall

import (
	"encoding/json"
	"monorepo/bin-chatbot-manager/models/chatbot"
	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines webhook event
type WebhookMessage struct {
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

	TMEnd    string `json:"tm_end,omitempty"`
	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts to the event
func (h *Chatbotcall) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		ChatbotID:          h.ChatbotID,
		ChatbotEngineType:  h.ChatbotEngineType,
		ChatbotEngineModel: h.ChatbotEngineModel,

		ActiveflowID:  h.ActiveflowID,
		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		ConfbridgeID: h.ConfbridgeID,
		TranscribeID: h.TranscribeID,

		Status: h.Status,

		Gender:   h.Gender,
		Language: h.Language,

		TMEnd:    h.TMEnd,
		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Chatbotcall) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

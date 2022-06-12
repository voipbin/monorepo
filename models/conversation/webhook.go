package conversation

import (
	"encoding/json"

	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID uuid.UUID `json:"id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   string        `json:"reference_id"`

	Participants []commonaddress.Address `json:"participants"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Conversation) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID: h.ID,

		Name:   h.Name,
		Detail: h.Detail,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		Participants: h.Participants,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Conversation) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

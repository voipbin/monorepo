package chatroom

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerType  OwnerType `json:"owner_type"`
	OwnerID    uuid.UUID `json:"owner_id"`

	Type   Type      `json:"type"`
	ChatID uuid.UUID `json:"chat_id"`

	RoomOwnerID    uuid.UUID   `json:"roomowner_id"`    // owned agent id
	ParticipantIDs []uuid.UUID `json:"participant_ids"` // list of participated ids(agent ids)

	Name   string `json:"name"`
	Detail string `json:"detail"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Chatroom) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,
		OwnerType:  h.OwnerType,
		OwnerID:    h.OwnerID,

		Type:   h.Type,
		ChatID: h.ChatID,

		RoomOwnerID:    h.RoomOwnerID,
		ParticipantIDs: h.ParticipantIDs,

		Name:   h.Name,
		Detail: h.Detail,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Chatroom) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

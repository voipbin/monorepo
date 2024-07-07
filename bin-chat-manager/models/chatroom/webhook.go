package chatroom

import (
	"encoding/json"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity
	commonidentity.Owner

	Type   Type      `json:"type"`
	ChatID uuid.UUID `json:"chat_id"`

	RoomOwnerID    uuid.UUID   `json:"room_owner_id"`   // owned agent id
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
		Identity: h.Identity,
		Owner:    h.Owner,

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

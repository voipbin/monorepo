package messagechatroom

import (
	"encoding/json"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-chat-manager/models/media"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	OwnerType  OwnerType `json:"owner_type"` //
	OwnerID    uuid.UUID `json:"owner_id"`

	ChatroomID    uuid.UUID `json:"chatroom_id"`
	MessagechatID uuid.UUID `json:"messagechat_id"`

	Source *commonaddress.Address `json:"source"`
	Type   Type                   `json:"type"`
	Text   string                 `json:"text"`
	Medias []media.Media          `json:"medias"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Messagechatroom) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,
		OwnerType:  h.OwnerType,
		OwnerID:    h.OwnerID,

		ChatroomID:    h.ChatroomID,
		MessagechatID: h.MessagechatID,

		Source: h.Source,
		Type:   h.Type,
		Text:   h.Text,
		Medias: h.Medias,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *Messagechatroom) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

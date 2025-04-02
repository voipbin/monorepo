package message

import (
	"encoding/json"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-message-manager/models/target"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	Type Type `json:"type"`

	// from/to info
	Source  *commonaddress.Address `json:"source"`
	Targets []target.Target        `json:"targets"`

	// message info
	Text      string    `json:"text"` // Text delivered in the body of the message.
	Direction Direction `json:"direction"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Message) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Type: h.Type,

		Source:  h.Source,
		Targets: h.Targets,

		Text:      h.Text,
		Direction: h.Direction,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Message) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

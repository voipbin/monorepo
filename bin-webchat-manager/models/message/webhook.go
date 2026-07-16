package message

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines the externally-visible shape of a Message,
// published both as the webhook payload and the
// EventTypeMessageCreated event (consumed by conversation-manager's
// §16 message-manager-pattern integration).
type WebhookMessage struct {
	commonidentity.Identity

	WidgetID  uuid.UUID `json:"widget_id"`
	SessionID uuid.UUID `json:"session_id"`

	Direction Direction `json:"direction"`
	Status    Status    `json:"status"`

	SenderID uuid.UUID `json:"sender_id,omitempty"`

	Text string `json:"text"`

	TMCreate *time.Time `json:"tm_create"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts a Message to its externally-visible
// WebhookMessage. ActiveflowID is intentionally NOT included — it is
// internal plumbing (the "already triggered" marker read by
// messagehandler.Create), not a visitor- or consumer-facing field.
func (h *Message) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		WidgetID:  h.WidgetID,
		SessionID: h.SessionID,

		Direction: h.Direction,
		Status:    h.Status,

		SenderID: h.SenderID,

		Text: h.Text,

		TMCreate: h.TMCreate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the marshaled webhook/event payload.
func (h *Message) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	res, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return res, nil
}

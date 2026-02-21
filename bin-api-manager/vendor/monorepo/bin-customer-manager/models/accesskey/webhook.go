package accesskey

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	Token       string `json:"token,omitempty"`
	TokenPrefix string `json:"token_prefix,omitempty"`

	TMExpire *time.Time `json:"tm_expire"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Accesskey) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		Name:   h.Name,
		Detail: h.Detail,

		Token:       h.RawToken,
		TokenPrefix: h.TokenPrefix,

		TMExpire: h.TMExpire,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Accesskey) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

package account

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Balance float32 `json:"balance"` // USD

	PaymentType   PaymentType   `json:"payment_type"`
	PaymentMethod PaymentMethod `json:"payment_method"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Account) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:         h.ID,
		CustomerID: h.CustomerID,

		Name:   h.Name,
		Detail: h.Detail,

		Balance: h.Balance,

		PaymentType:   h.PaymentType,
		PaymentMethod: h.PaymentMethod,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Account) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

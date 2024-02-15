package customer

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	ID uuid.UUID `json:"id"` // Customer's ID

	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail

	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Address     string `json:"address,omitempty"`

	// webhook info
	WebhookMethod WebhookMethod `json:"webhook_method"` // webhook method
	WebhookURI    string        `json:"webhook_uri"`    // webhook uri

	BillingAccountID uuid.UUID `json:"billing_account_id"` // default billing account id

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// ConvertWebhookMessage converts to the event
func (h *Customer) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID: h.ID,

		Name:   h.Name,
		Detail: h.Detail,

		Email:       h.Email,
		PhoneNumber: h.PhoneNumber,
		Address:     h.Address,

		WebhookMethod: h.WebhookMethod,
		WebhookURI:    h.WebhookURI,

		BillingAccountID: h.BillingAccountID,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}

}

// CreateWebhookEvent generate WebhookEvent
func (h *Customer) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

package availablenumber

import "encoding/json"

// WebhookMessage defines
type WebhookMessage struct {
	Number string `json:"number"`

	Country    string    `json:"country"`
	Region     string    `json:"region"`
	PostalCode string    `json:"postal_code"`
	Features   []Feature `json:"features"`
}

// ConvertWebhookMessage converts to the event
func (h *AvailableNumber) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Number: h.Number,

		Country:    h.Country,
		Region:     h.Region,
		PostalCode: h.PostalCode,
		Features:   h.Features,
	}
}

// CreateWebhookEvent generates the WebhookEvent
func (h *AvailableNumber) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

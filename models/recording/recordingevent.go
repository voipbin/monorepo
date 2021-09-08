package recording

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// WebhookEvent defines recording event for webhook
type WebhookEvent struct {
	Type string           `json:"type"` // event type
	Data WebhookEventData `json:"data"`
}

// WebhookEventData struct represent record information
type WebhookEventData struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	ReferenceID uuid.UUID `json:"reference_id"`
	Status      string    `json:"status"`
	Format      string    `json:"format"`
	WebhookURI  string    `json:"webhook_uri"`

	TMStart string `json:"tm_start"`
	TMEnd   string `json:"tm_end"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// CreateWebhookEvent generates webhook event
func (h *Recording) CreateWebhookEvent(t string) ([]byte, error) {
	e := &WebhookEvent{
		Type: t,
		Data: WebhookEventData{
			ID:          h.ID,
			Type:        string(h.Type),
			ReferenceID: h.ReferenceID,
			Status:      string(h.Status),
			Format:      h.Format,
			WebhookURI:  h.WebhookURI,
			TMStart:     h.TMStart,
			TMEnd:       h.TMEnd,
			TMCreate:    h.TMCreate,
			TMUpdate:    h.TMUpdate,
			TMDelete:    h.TMDelete,
		},
	}

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil

}

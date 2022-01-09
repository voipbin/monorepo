package recording

import (
	"encoding/json"

	"github.com/gofrs/uuid"
)

// Event struct represent record information
type Event struct {
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

// ConvertEvent converts to the event
func (h *Recording) ConvertEvent() *Event {
	return &Event{
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
	}
}

// CreateWebhookEvent generates webhook event
func (h *Recording) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertEvent()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil

}

package recording

import (
	"encoding/json"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage struct represent record information
type WebhookMessage struct {
	commonidentity.Identity
	commonidentity.Owner

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`
	Status        Status        `json:"status"`
	Format        Format        `json:"format"`

	TMStart string `json:"tm_start"`
	TMEnd   string `json:"tm_end"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Recording) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,
		Owner:    h.Owner,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,
		Status:        h.Status,
		Format:        h.Format,

		TMStart: h.TMStart,
		TMEnd:   h.TMEnd,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates webhook event
func (h *Recording) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil

}

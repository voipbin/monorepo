package conferencecall

import (
	"encoding/json"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines conferencecall webhook event
type WebhookMessage struct {
	commonidentity.Identity

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
	ConferenceID uuid.UUID `json:"conference_id,omitempty"`

	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	Status Status `json:"status,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts to the event
func (h *Conferencecall) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		ActiveflowID: h.ActiveflowID,
		ConferenceID: h.ConferenceID,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		Status: h.Status,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Conferencecall) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

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

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty"`
	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`
	Status        Status        `json:"status,omitempty"`
	Format        Format        `json:"format,omitempty"`

	OnEndFlowID uuid.UUID `json:"on_end_flow_id,omitempty"` // executed when recording ends

	TMStart string `json:"tm_start,omitempty"`
	TMEnd   string `json:"tm_end,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts to the event
func (h *Recording) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,
		Owner:    h.Owner,

		ActiveflowID:  h.ActiveflowID,
		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,
		Status:        h.Status,
		Format:        h.Format,

		OnEndFlowID: h.OnEndFlowID,

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

package queuecall

import (
	"encoding/json"
	commonidentity "monorepo/bin-common-handler/models/identity"

	uuid "github.com/gofrs/uuid"
)

// WebhookMessage defines conference webhook event
type WebhookMessage struct {
	commonidentity.Identity

	ReferenceType string    `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID `json:"reference_id,omitempty"`

	Status         string    `json:"status,omitempty"`
	ServiceAgentID uuid.UUID `json:"service_agent_id,omitempty"`

	DurationWaiting int `json:"duration_waiting,omitempty"` // duration for waiting(ms)
	DurationService int `json:"duration_service,omitempty"` // duration for service(ms)

	TMCreate  string `json:"tm_create,omitempty"`
	TMService string `json:"tm_service,omitempty"`
	TMUpdate  string `json:"tm_update,omitempty"`
	TMDelete  string `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage defines
func (h *Queuecall) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		ReferenceType: string(h.ReferenceType),
		ReferenceID:   h.ReferenceID,

		Status:         string(h.Status),
		ServiceAgentID: h.ServiceAgentID,

		DurationWaiting: h.DurationWaiting,
		DurationService: h.DurationService,

		TMCreate:  h.TMCreate,
		TMService: h.TMService,
		TMUpdate:  h.TMUpdate,
		TMDelete:  h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Queuecall) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

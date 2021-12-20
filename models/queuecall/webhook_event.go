package queuecall

import (
	"encoding/json"

	uuid "github.com/gofrs/uuid"
)

// WebhookEventData defines conference webhook event
type WebhookEventData struct {
	ID            uuid.UUID `json:"id"`
	ReferenceType string    `json:"reference_type"`
	ReferenceID   uuid.UUID `json:"reference_id"`

	Status         string    `json:"status"`
	ServiceAgentID uuid.UUID `json:"service_agent_id"`

	TMCreate  string `json:"tm_create"`
	TMService string `json:"tm_service"`
	TMUpdate  string `json:"tm_update"`
	TMDelete  string `json:"tm_delete"`
}

// CreateWebhookEvent generate WebhookEvent
func (h *Queuecall) CreateWebhookEvent(t string) ([]byte, error) {
	e := &WebhookEventData{
		ID:            h.ID,
		ReferenceType: string(h.ReferenceType),
		ReferenceID:   h.ReferenceID,

		Status:         string(h.Status),
		ServiceAgentID: h.ServiceAgentID,

		TMCreate:  h.TMCreate,
		TMService: h.TMService,
		TMUpdate:  h.TMUpdate,
		TMDelete:  h.TMDelete,
	}

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

package queuecall

import (
	"encoding/json"

	uuid "github.com/gofrs/uuid"
)

// Event defines conference webhook event
type Event struct {
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

// ConvertEvent defines
func (h *Queuecall) ConvertEvent() *Event {
	return &Event{
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
}

// CreateWebhookEvent generate WebhookEvent
func (h *Queuecall) CreateWebhookEvent(t string) ([]byte, error) {
	e := h.ConvertEvent()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

package rag

import (
	"time"

	"github.com/gofrs/uuid"
)

// WebhookMessage is the external-facing representation of a Rag
type WebhookMessage struct {
	ID          uuid.UUID  `json:"id,omitempty"`
	CustomerID  uuid.UUID  `json:"customer_id,omitempty"`
	Name        string     `json:"name,omitempty"`
	Description string     `json:"description,omitempty"`
	TMCreate    *time.Time `json:"tm_create,omitempty"`
	TMUpdate    *time.Time `json:"tm_update,omitempty"`
	TMDelete    *time.Time `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts the internal Rag to the external representation
func (r *Rag) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:          r.ID,
		CustomerID:  r.CustomerID,
		Name:        r.Name,
		Description: r.Description,
		TMCreate:    r.TMCreate,
		TMUpdate:    r.TMUpdate,
		TMDelete:    r.TMDelete,
	}
}

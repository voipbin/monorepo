package rag

import (
	"time"

	"github.com/gofrs/uuid"

	rmdocument "monorepo/bin-rag-manager/models/document"
)

// WebhookMessage is the external-facing representation of a RAG.
type WebhookMessage struct {
	ID          uuid.UUID         `json:"id,omitempty"`
	CustomerID  uuid.UUID         `json:"customer_id,omitempty"`
	Name        string            `json:"name,omitempty"`
	Description string            `json:"description,omitempty"`
	Status      rmdocument.Status `json:"status,omitempty"`
	Sources     []Source          `json:"sources,omitempty"`
	TMCreate    *time.Time        `json:"tm_create,omitempty"`
	TMUpdate    *time.Time        `json:"tm_update,omitempty"`
}

// ConvertWebhookMessage converts internal Rag to external representation.
// Status and Sources are copied from transient fields (populated by raghandler).
func (r *Rag) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:          r.ID,
		CustomerID:  r.CustomerID,
		Name:        r.Name,
		Description: r.Description,
		Status:      r.Status,
		Sources:     r.Sources,
		TMCreate:    r.TMCreate,
		TMUpdate:    r.TMUpdate,
	}
}

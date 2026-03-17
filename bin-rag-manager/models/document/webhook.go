package document

import (
	"time"

	"github.com/gofrs/uuid"
)

// WebhookMessage is the external-facing representation of a Document.
// TMDelete is omitted — soft-deleted records are not returned via API.
type WebhookMessage struct {
	ID            uuid.UUID  `json:"id,omitempty"`
	CustomerID    uuid.UUID  `json:"customer_id,omitempty"`
	RagID         uuid.UUID  `json:"rag_id,omitempty"`
	Name          string     `json:"name,omitempty"`
	DocType       DocType    `json:"doc_type,omitempty"`
	StorageFileID uuid.UUID  `json:"storage_file_id,omitempty"`
	SourceURL     string     `json:"source_url,omitempty"`
	Status        Status     `json:"status,omitempty"`
	StatusMessage string     `json:"status_message,omitempty"`
	TMCreate      *time.Time `json:"tm_create,omitempty"`
	TMUpdate      *time.Time `json:"tm_update,omitempty"`
}

// ConvertWebhookMessage converts the internal Document to the external representation
func (d *Document) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		ID:            d.ID,
		CustomerID:    d.CustomerID,
		RagID:         d.RagID,
		Name:          d.Name,
		DocType:       d.DocType,
		StorageFileID: d.StorageFileID,
		SourceURL:     d.SourceURL,
		Status:        d.Status,
		StatusMessage: d.StatusMessage,
		TMCreate:      d.TMCreate,
		TMUpdate:      d.TMUpdate,
	}
}

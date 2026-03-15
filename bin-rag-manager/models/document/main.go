package document

import (
	"time"

	"github.com/gofrs/uuid"
)

// DocType defines the document source type
type DocType string

const (
	DocTypeUploaded  DocType = "uploaded"
	DocTypeURL       DocType = "url"
	DocTypePlatform  DocType = "platform"
	DocTypeGenerated DocType = "generated"
)

// Status defines the document processing status
type Status string

const (
	StatusPending    Status = "pending"
	StatusProcessing Status = "processing"
	StatusReady      Status = "ready"
	StatusError      Status = "error"
)

// Document represents a document within a RAG
type Document struct {
	ID            uuid.UUID  `json:"id,omitempty" db:"id,uuid"`
	CustomerID    uuid.UUID  `json:"customer_id,omitempty" db:"customer_id,uuid"`
	RagID         uuid.UUID  `json:"rag_id,omitempty" db:"rag_id,uuid"`
	Name          string     `json:"name,omitempty" db:"name"`
	DocType       DocType    `json:"doc_type,omitempty" db:"doc_type"`
	StorageFileID uuid.UUID  `json:"storage_file_id,omitempty" db:"storage_file_id,uuid"`
	SourceURL     string     `json:"source_url,omitempty" db:"source_url"`
	Status        Status     `json:"status,omitempty" db:"status"`
	StatusMessage string     `json:"status_message,omitempty" db:"status_message"`
	TMCreate      *time.Time `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate      *time.Time `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete      *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
}

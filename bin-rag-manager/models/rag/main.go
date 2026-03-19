package rag

import (
	"time"

	"github.com/gofrs/uuid"

	rmdocument "monorepo/bin-rag-manager/models/document"
)

// Rag represents a knowledge base container
type Rag struct {
	ID          uuid.UUID  `json:"id,omitempty" db:"id,uuid"`
	CustomerID  uuid.UUID  `json:"customer_id,omitempty" db:"customer_id,uuid"`
	Name        string     `json:"name,omitempty" db:"name"`
	Description string     `json:"description,omitempty" db:"description"`
	TMCreate    *time.Time `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate    *time.Time `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete    *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`

	// Transient — populated by handler, ignored by DB (no db tag)
	Status  rmdocument.Status `json:"status,omitempty"`
	Sources []Source          `json:"sources,omitempty"`
}

// Source represents a single source (document) in the RAG response.
type Source struct {
	StorageFileID *uuid.UUID        `json:"storage_file_id,omitempty"`
	SourceURL     string            `json:"source_url,omitempty"`
	Status        rmdocument.Status `json:"status,omitempty"`
	StatusMessage string            `json:"status_message,omitempty"`
}

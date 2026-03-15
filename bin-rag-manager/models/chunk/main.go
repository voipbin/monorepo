package chunk

import (
	"time"

	"github.com/gofrs/uuid"
)

// Chunk represents a text chunk with its embedding vector
type Chunk struct {
	ID           uuid.UUID  `json:"id,omitempty" db:"id,uuid"`
	DocumentID   uuid.UUID  `json:"document_id,omitempty" db:"document_id,uuid"`
	RagID        uuid.UUID  `json:"rag_id,omitempty" db:"rag_id,uuid"`
	CustomerID   uuid.UUID  `json:"customer_id,omitempty" db:"customer_id,uuid"`
	ChunkIndex   int        `json:"chunk_index,omitempty" db:"chunk_index"`
	Text         string     `json:"text,omitempty" db:"text"`
	SectionTitle string     `json:"section_title,omitempty" db:"section_title"`
	Embedding    []float32  `json:"-" db:"-"`
	TokenCount   int        `json:"token_count,omitempty" db:"token_count"`
	TMCreate     *time.Time `json:"tm_create,omitempty" db:"tm_create"`
	TMDelete     *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
}

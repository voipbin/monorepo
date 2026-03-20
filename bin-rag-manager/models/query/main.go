package query

import (
	"github.com/gofrs/uuid"
)

// Request represents a RAG query request
type Request struct {
	RagID uuid.UUID `json:"rag_id"`
	Query string    `json:"query"`
	TopK  int       `json:"top_k,omitempty"`
}

// Source represents a source reference in the query response
type Source struct {
	DocumentID     uuid.UUID `json:"document_id"`
	DocumentName   string    `json:"document_name"`
	SectionTitle   string    `json:"section_title"`
	RelevanceScore float64   `json:"relevance_score"`
	Text           string    `json:"text"`
}

// Response represents a RAG query response — sources only.
// Answer generation is handled by the caller (e.g., pipecat-manager).
type Response struct {
	Sources []Source `json:"sources"`
}

package retriever

import (
	"context"
	"fmt"

	"monorepo/bin-rag-manager/pkg/chunker"
	"monorepo/bin-rag-manager/pkg/embedder"
	"monorepo/bin-rag-manager/pkg/store"
)

// Retriever defines the interface for document retrieval
type Retriever interface {
	Query(ctx context.Context, query string, topK int, docTypes []chunker.DocType) ([]store.SearchResult, error)
}

type retriever struct {
	embedder embedder.Embedder
	store    store.Store
}

// NewRetriever creates a new Retriever
func NewRetriever(emb embedder.Embedder, st store.Store) Retriever {
	return &retriever{
		embedder: emb,
		store:    st,
	}
}

// Query embeds the query and performs similarity search
func (r *retriever) Query(ctx context.Context, query string, topK int, docTypes []chunker.DocType) ([]store.SearchResult, error) {
	queryEmbedding, err := r.embedder.EmbedText(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("could not embed query: %w", err)
	}

	results := r.store.Search(queryEmbedding, topK, docTypes)
	return results, nil
}

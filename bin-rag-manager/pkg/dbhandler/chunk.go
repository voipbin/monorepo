package dbhandler

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/chunk"
)

// ChunkCreate inserts a new chunk with its embedding vector
func (h *handler) ChunkCreate(ctx context.Context, c *chunk.Chunk, embedding []float32) error {
	return fmt.Errorf("not implemented")
}

// ChunkCreateBatch inserts multiple chunks with their embedding vectors
func (h *handler) ChunkCreateBatch(ctx context.Context, chunks []*chunk.Chunk, embeddings [][]float32) error {
	return fmt.Errorf("not implemented")
}

// ChunkSearchByRagID performs vector similarity search within a rag
func (h *handler) ChunkSearchByRagID(ctx context.Context, ragID uuid.UUID, queryEmbedding []float32, topK int) ([]*chunk.Chunk, []float64, error) {
	return nil, nil, fmt.Errorf("not implemented")
}

// ChunkDeleteByDocumentID hard-deletes all chunks for a document
func (h *handler) ChunkDeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error {
	return fmt.Errorf("not implemented")
}

// ChunkDeleteByRagID hard-deletes all chunks for a rag
func (h *handler) ChunkDeleteByRagID(ctx context.Context, ragID uuid.UUID) error {
	return fmt.Errorf("not implemented")
}

// ChunkSoftDeleteByDocumentID soft-deletes all chunks for a document
func (h *handler) ChunkSoftDeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error {
	return fmt.Errorf("not implemented")
}

// ChunkSoftDeleteByRagID soft-deletes all chunks for a rag
func (h *handler) ChunkSoftDeleteByRagID(ctx context.Context, ragID uuid.UUID) error {
	return fmt.Errorf("not implemented")
}

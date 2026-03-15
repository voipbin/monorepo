package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"

	sq "github.com/Masterminds/squirrel"
	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/chunk"
	"monorepo/bin-rag-manager/models/document"
	"monorepo/bin-rag-manager/models/rag"
)

// psql is a squirrel StatementBuilder configured for PostgreSQL dollar placeholders.
// Shared across all dbhandler files (rag.go, document.go, chunk.go).
var psql = sq.StatementBuilder.PlaceholderFormat(sq.Dollar)

// DBHandler defines all database operations for rag-manager
type DBHandler interface {
	// Rag operations
	RagCreate(ctx context.Context, r *rag.Rag) error
	RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error)
	RagGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*rag.Rag, error)
	RagUpdate(ctx context.Context, id uuid.UUID, fields map[rag.Field]any) error
	RagDelete(ctx context.Context, id uuid.UUID) error

	// Document operations
	DocumentCreate(ctx context.Context, d *document.Document) error
	DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error)
	DocumentGetsByRagID(ctx context.Context, ragID uuid.UUID) ([]*document.Document, error)
	DocumentGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*document.Document, error)
	DocumentUpdate(ctx context.Context, id uuid.UUID, fields map[document.Field]any) error
	DocumentDelete(ctx context.Context, id uuid.UUID) error
	DocumentDeleteByRagID(ctx context.Context, ragID uuid.UUID) error

	// Chunk operations
	ChunkCreate(ctx context.Context, c *chunk.Chunk, embedding []float32) error
	ChunkCreateBatch(ctx context.Context, chunks []*chunk.Chunk, embeddings [][]float32) error
	ChunkSearchByRagID(ctx context.Context, ragID uuid.UUID, queryEmbedding []float32, topK int) ([]*chunk.Chunk, []float64, error)
	ChunkDeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error
	ChunkDeleteByRagID(ctx context.Context, ragID uuid.UUID) error
	ChunkSoftDeleteByDocumentID(ctx context.Context, documentID uuid.UUID) error
	ChunkSoftDeleteByRagID(ctx context.Context, ragID uuid.UUID) error
}

// handler implements DBHandler using PostgreSQL
type handler struct {
	db *sql.DB
}

// NewHandler creates a new DBHandler
func NewHandler(db *sql.DB) DBHandler {
	return &handler{
		db: db,
	}
}

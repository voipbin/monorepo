package raghandler

//go:generate mockgen -package raghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/document"
	"monorepo/bin-rag-manager/models/query"
	"monorepo/bin-rag-manager/models/rag"
	"monorepo/bin-rag-manager/pkg/dbhandler"
	"monorepo/bin-rag-manager/pkg/embedder"
)

// RagHandler defines the interface for RAG operations
type RagHandler interface {
	RagCreate(ctx context.Context, customerID uuid.UUID, name, description string) (*rag.Rag, error)
	RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error)
	RagList(ctx context.Context, size uint64, token string, filters map[rag.Field]any) ([]*rag.Rag, error)
	RagUpdate(ctx context.Context, id uuid.UUID, fields map[rag.Field]any) (*rag.Rag, error)
	RagDelete(ctx context.Context, id uuid.UUID) error

	DocumentCreate(ctx context.Context, customerID, ragID uuid.UUID, name string, docType document.DocType, sourceURL string, storageFileID uuid.UUID) (*document.Document, error)
	DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error)
	DocumentList(ctx context.Context, size uint64, token string, filters map[document.Field]any) ([]*document.Document, error)
	DocumentDelete(ctx context.Context, id uuid.UUID) error

	QueryRag(ctx context.Context, ragID uuid.UUID, queryText string, topK int) (*query.Response, error)
}

type ragHandler struct {
	embedder  embedder.Embedder
	dbHandler dbhandler.DBHandler
}

// NewRagHandler creates a new RagHandler
func NewRagHandler(
	emb embedder.Embedder,
	dbH dbhandler.DBHandler,
) RagHandler {
	return &ragHandler{
		embedder:  emb,
		dbHandler: dbH,
	}
}

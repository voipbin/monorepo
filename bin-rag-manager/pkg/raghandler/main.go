package raghandler

//go:generate mockgen -package raghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"fmt"

	"github.com/gofrs/uuid"

	"monorepo/bin-rag-manager/models/document"
	"monorepo/bin-rag-manager/models/query"
	"monorepo/bin-rag-manager/models/rag"
	"monorepo/bin-rag-manager/pkg/dbhandler"
	"monorepo/bin-rag-manager/pkg/embedder"
)

// RagHandler defines the interface for RAG operations
type RagHandler interface {
	RagCreate(ctx context.Context, customerID uuid.UUID, name, description string, fileIDs []uuid.UUID, urls []string) (*rag.Rag, error)
	RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error)
	RagGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*rag.Rag, error)
	RagDelete(ctx context.Context, id uuid.UUID) error

	DocumentCreate(ctx context.Context, customerID, ragID uuid.UUID, fileIDs []uuid.UUID, urls []string) ([]*document.Document, error)
	DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error)
	DocumentGetsByRagID(ctx context.Context, ragID uuid.UUID) ([]*document.Document, error)
	DocumentGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*document.Document, error)
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

// RagCreate creates a new RAG configuration for a customer
func (h *ragHandler) RagCreate(ctx context.Context, customerID uuid.UUID, name, description string, fileIDs []uuid.UUID, urls []string) (*rag.Rag, error) {
	return nil, fmt.Errorf("not implemented")
}

// RagGet retrieves a RAG configuration by ID
func (h *ragHandler) RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error) {
	return nil, fmt.Errorf("not implemented")
}

// RagGetsByCustomerID retrieves all RAG configurations for a customer
func (h *ragHandler) RagGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*rag.Rag, error) {
	return nil, fmt.Errorf("not implemented")
}

// RagDelete deletes a RAG configuration by ID
func (h *ragHandler) RagDelete(ctx context.Context, id uuid.UUID) error {
	return fmt.Errorf("not implemented")
}

// DocumentCreate creates documents for a RAG configuration
func (h *ragHandler) DocumentCreate(ctx context.Context, customerID, ragID uuid.UUID, fileIDs []uuid.UUID, urls []string) ([]*document.Document, error) {
	return nil, fmt.Errorf("not implemented")
}

// DocumentGet retrieves a document by ID
func (h *ragHandler) DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error) {
	return nil, fmt.Errorf("not implemented")
}

// DocumentGetsByRagID retrieves all documents for a RAG configuration
func (h *ragHandler) DocumentGetsByRagID(ctx context.Context, ragID uuid.UUID) ([]*document.Document, error) {
	return nil, fmt.Errorf("not implemented")
}

// DocumentGetsByCustomerID retrieves all documents for a customer
func (h *ragHandler) DocumentGetsByCustomerID(ctx context.Context, customerID uuid.UUID) ([]*document.Document, error) {
	return nil, fmt.Errorf("not implemented")
}

// DocumentDelete deletes a document by ID
func (h *ragHandler) DocumentDelete(ctx context.Context, id uuid.UUID) error {
	return fmt.Errorf("not implemented")
}

// QueryRag performs a RAG query against a specific RAG configuration
func (h *ragHandler) QueryRag(ctx context.Context, ragID uuid.UUID, queryText string, topK int) (*query.Response, error) {
	return nil, fmt.Errorf("not implemented")
}

package raghandler

//go:generate mockgen -package raghandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"github.com/gofrs/uuid"

	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-rag-manager/models/document"
	"monorepo/bin-rag-manager/models/query"
	"monorepo/bin-rag-manager/models/rag"
	"monorepo/bin-rag-manager/pkg/bucketreader"
	"monorepo/bin-rag-manager/pkg/dbhandler"
	"monorepo/bin-rag-manager/pkg/embedder"
)

// RagHandler defines the interface for RAG operations
type RagHandler interface {
	RagCreate(ctx context.Context, customerID uuid.UUID, name, description string, storageFileIDs []uuid.UUID, sourceURLs []string) (*rag.Rag, error)
	RagGet(ctx context.Context, id uuid.UUID) (*rag.Rag, error)
	RagList(ctx context.Context, size uint64, token string, filters map[rag.Field]any) ([]*rag.Rag, error)
	RagUpdate(ctx context.Context, id uuid.UUID, fields map[rag.Field]any) (*rag.Rag, error)
	RagDelete(ctx context.Context, id uuid.UUID) error
	RagAddSources(ctx context.Context, ragID uuid.UUID, storageFileIDs []uuid.UUID, sourceURLs []string) (*rag.Rag, error)

	DocumentGet(ctx context.Context, id uuid.UUID) (*document.Document, error)
	DocumentList(ctx context.Context, size uint64, token string, filters map[document.Field]any) ([]*document.Document, error)

	QueryRag(ctx context.Context, ragID uuid.UUID, queryText string, topK int) (*query.Response, error)

	DocumentIngestPendingAll(ctx context.Context)
	RunIngestionTicker(ctx context.Context, interval time.Duration)
}

// maxConcurrentIngestions limits how many documents can be ingested simultaneously
// to avoid overwhelming the embedding API, file descriptors, and memory.
const maxConcurrentIngestions = 5

type ragHandler struct {
	embedder     embedder.Embedder
	dbHandler    dbhandler.DBHandler
	reqHandler   requesthandler.RequestHandler
	bucketReader bucketreader.BucketReader
	bucketName   string
	ingestSem    chan struct{} // semaphore to limit concurrent ingestion goroutines
}

// NewRagHandler creates a new RagHandler
func NewRagHandler(
	emb embedder.Embedder,
	dbH dbhandler.DBHandler,
	reqH requesthandler.RequestHandler,
	br bucketreader.BucketReader,
	bucketName string,
) RagHandler {
	return &ragHandler{
		embedder:     emb,
		dbHandler:    dbH,
		reqHandler:   reqH,
		bucketReader: br,
		bucketName:   bucketName,
		ingestSem:    make(chan struct{}, maxConcurrentIngestions),
	}
}

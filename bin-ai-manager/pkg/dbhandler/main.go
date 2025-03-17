package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-ai-manager/models/aicall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	AICreate(ctx context.Context, c *ai.AI) error
	AIDelete(ctx context.Context, id uuid.UUID) error
	AIGet(ctx context.Context, id uuid.UUID) (*ai.AI, error)
	AIGets(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*ai.AI, error)
	AISetInfo(ctx context.Context, id uuid.UUID, name string, detail string, engineType ai.EngineType, engineModel ai.EngineModel, engineData map[string]any, initPrompt string) error

	AIcallCreate(ctx context.Context, cb *aicall.AIcall) error
	AIcallDelete(ctx context.Context, id uuid.UUID) error
	AIcallGet(ctx context.Context, id uuid.UUID) (*aicall.AIcall, error)
	AIcallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*aicall.AIcall, error)
	AIcallGetByTranscribeID(ctx context.Context, transcribeID uuid.UUID) (*aicall.AIcall, error)
	AIcallGets(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*aicall.AIcall, error)
	AIcallUpdateStatusProgressing(ctx context.Context, id uuid.UUID, transcribeID uuid.UUID) error
	AIcallUpdateStatusEnd(ctx context.Context, id uuid.UUID) error

	MessageCreate(ctx context.Context, c *message.Message) error
	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageGets(ctx context.Context, aicallID uuid.UUID, size uint64, token string, filters map[string]string) ([]*message.Message, error)
	MessageDelete(ctx context.Context, id uuid.UUID) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("record not found")
)

// list of default values
const (
	DefaultTimeStamp = "9999-01-01 00:00:00.000000" //nolint:varcheck,deadcode // this is fine
)

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
	return h
}

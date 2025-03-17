package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-ai-manager/models/chatbot"
	"monorepo/bin-ai-manager/models/chatbotcall"
	"monorepo/bin-ai-manager/models/message"
	"monorepo/bin-ai-manager/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	ChatbotCreate(ctx context.Context, c *chatbot.Chatbot) error
	ChatbotDelete(ctx context.Context, id uuid.UUID) error
	ChatbotGet(ctx context.Context, id uuid.UUID) (*chatbot.Chatbot, error)
	ChatbotGets(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*chatbot.Chatbot, error)
	ChatbotSetInfo(ctx context.Context, id uuid.UUID, name string, detail string, engineType chatbot.EngineType, engineModel chatbot.EngineModel, engineData map[string]any, initPrompt string) error

	ChatbotcallCreate(ctx context.Context, cb *chatbotcall.Chatbotcall) error
	ChatbotcallDelete(ctx context.Context, id uuid.UUID) error
	ChatbotcallGet(ctx context.Context, id uuid.UUID) (*chatbotcall.Chatbotcall, error)
	ChatbotcallGetByReferenceID(ctx context.Context, referenceID uuid.UUID) (*chatbotcall.Chatbotcall, error)
	ChatbotcallGetByTranscribeID(ctx context.Context, transcribeID uuid.UUID) (*chatbotcall.Chatbotcall, error)
	ChatbotcallGets(ctx context.Context, customerID uuid.UUID, size uint64, token string, filters map[string]string) ([]*chatbotcall.Chatbotcall, error)
	ChatbotcallUpdateStatusProgressing(ctx context.Context, id uuid.UUID, transcribeID uuid.UUID) error
	ChatbotcallUpdateStatusEnd(ctx context.Context, id uuid.UUID) error

	MessageCreate(ctx context.Context, c *message.Message) error
	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageGets(ctx context.Context, chatbotcallID uuid.UUID, size uint64, token string, filters map[string]string) ([]*message.Message, error)
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

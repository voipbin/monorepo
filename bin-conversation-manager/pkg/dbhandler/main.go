package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/account"
	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	AccountCreate(ctx context.Context, ac *account.Account) error
	AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error)
	AccountGets(ctx context.Context, size uint64, token string, filters map[string]string) ([]*account.Account, error)
	AccountSet(ctx context.Context, id uuid.UUID, name string, detail string, secret string, token string) error
	AccountDelete(ctx context.Context, id uuid.UUID) error

	ConversationCreate(ctx context.Context, cv *conversation.Conversation) error
	ConversationGet(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error)
	// ConversationGetByTypeAndDialogID(ctx context.Context, conversationType conversation.Type, dialogID string) (*conversation.Conversation, error)
	ConversationGetBySelfAndPeer(ctx context.Context, self commonaddress.Address, peer commonaddress.Address) (*conversation.Conversation, error)
	ConversationGets(ctx context.Context, size uint64, token string, filters map[string]any) ([]*conversation.Conversation, error)
	ConversationSet(ctx context.Context, id uuid.UUID, name string, detail string) error

	MediaCreate(ctx context.Context, m *media.Media) error
	MediaGet(ctx context.Context, id uuid.UUID) (*media.Media, error)

	MessageCreate(ctx context.Context, m *message.Message) error
	MessageDelete(ctx context.Context, id uuid.UUID) error
	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageGets(ctx context.Context, token string, size uint64, filters map[string]string) ([]*message.Message, error)
	MessageGetsByTransactionID(ctx context.Context, transactionID string, token string, limit uint64) ([]*message.Message, error)
	MessageUpdateStatus(ctx context.Context, id uuid.UUID, status message.Status) error
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
	DefaultTimeStamp = "9999-01-01 00:00:000"
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

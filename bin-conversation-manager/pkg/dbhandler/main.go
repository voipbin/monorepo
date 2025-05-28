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
	AccountUpdate(ctx context.Context, id uuid.UUID, fields map[account.Field]any) error
	AccountGets(context.Context, uint64, string, map[account.Field]any) ([]*account.Account, error)
	AccountSet(ctx context.Context, id uuid.UUID, name string, detail string, secret string, token string) error
	AccountDelete(ctx context.Context, id uuid.UUID) error

	ConversationCreate(ctx context.Context, cv *conversation.Conversation) error
	ConversationGet(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error)
	ConversationGetBySelfAndPeer(ctx context.Context, self commonaddress.Address, peer commonaddress.Address) (*conversation.Conversation, error)
	ConversationGets(ctx context.Context, size uint64, token string, filters map[conversation.Field]any) ([]*conversation.Conversation, error)
	ConversationUpdate(ctx context.Context, id uuid.UUID, fields map[conversation.Field]any) error

	MediaCreate(ctx context.Context, m *media.Media) error
	MediaGet(ctx context.Context, id uuid.UUID) (*media.Media, error)

	MessageCreate(ctx context.Context, m *message.Message) error
	MessageDelete(ctx context.Context, id uuid.UUID) error
	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageGets(ctx context.Context, token string, size uint64, filters map[message.Field]any) ([]*message.Message, error)
	MessageGetsByTransactionID(ctx context.Context, transactionID string, token string, limit uint64) ([]*message.Message, error)
	MessageUpdate(ctx context.Context, id uuid.UUID, fields map[message.Field]any) error
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

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
	return h
}

package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/account"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/media"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
	"gitlab.com/voipbin/bin-manager/conversation-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	AccountCreate(ctx context.Context, ac *account.Account) error
	AccountGet(ctx context.Context, id uuid.UUID) (*account.Account, error)
	AccountGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*account.Account, error)
	AccountSet(ctx context.Context, id uuid.UUID, name string, detail string, secret string, token string) error
	AccountDelete(ctx context.Context, id uuid.UUID) error

	ConversationCreate(ctx context.Context, cv *conversation.Conversation) error
	ConversationGet(ctx context.Context, id uuid.UUID) (*conversation.Conversation, error)
	ConversationGetByReferenceInfo(ctx context.Context, customerID uuid.UUID, referenceType conversation.ReferenceType, referenceID string) (*conversation.Conversation, error)
	ConversationGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*conversation.Conversation, error)
	ConversationSet(ctx context.Context, id uuid.UUID, name string, detail string) error

	MediaCreate(ctx context.Context, m *media.Media) error
	MediaGet(ctx context.Context, id uuid.UUID) (*media.Media, error)

	MessageCreate(ctx context.Context, m *message.Message) error
	MessageDelete(ctx context.Context, id uuid.UUID) error
	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageGetsByConversationID(ctx context.Context, conversationID uuid.UUID, token string, limit uint64) ([]*message.Message, error)
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

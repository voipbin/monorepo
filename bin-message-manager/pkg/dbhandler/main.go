package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-message-manager/models/message"
	"monorepo/bin-message-manager/models/target"
	"monorepo/bin-message-manager/pkg/cachehandler"
)

// DBHandler interface for database handle
type DBHandler interface {
	MessageCreate(ctx context.Context, n *message.Message) error
	MessageDelete(ctx context.Context, id uuid.UUID) error
	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageList(ctx context.Context, token string, size uint64, filters map[message.Field]any) ([]*message.Message, error)
	MessageUpdate(ctx context.Context, id uuid.UUID, fields map[message.Field]any) error
	MessageUpdateTargets(ctx context.Context, id uuid.UUID, provider message.ProviderName, targets []target.Target) error

	// MessageGetFromDB returns the message from the database and bypasses the cache.
	MessageGetFromDB(ctx context.Context, id uuid.UUID) (*message.Message, error)
	// MessageListByProviderReference returns the outbound messages created since the given time whose targets contain the
	// text of the provider reference id. The caller compares the exact id of each target.
	MessageListByProviderReference(ctx context.Context, ref string, since time.Time) ([]*message.Message, error)
	// MessageUpdateTargetsIf writes the targets (and the provider name if given) only if tm_update still equals prevTMUpdate
	// (NULL when the message was never updated). It returns false when no row matched, which is a concurrent write.
	MessageUpdateTargetsIf(ctx context.Context, id uuid.UUID, prevTMUpdate *time.Time, provider *message.ProviderName, targets []target.Target) (bool, error)
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

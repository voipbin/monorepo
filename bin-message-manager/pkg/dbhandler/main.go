package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

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
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// List of default values
const (
	DefaultTimeStamp = "9999-01-01T00:00:00.000000Z"
)

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

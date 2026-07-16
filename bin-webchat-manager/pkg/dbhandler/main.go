package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	context "context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	uuid "github.com/gofrs/uuid"

	"monorepo/bin-webchat-manager/models/message"
	"monorepo/bin-webchat-manager/models/session"
	"monorepo/bin-webchat-manager/models/widget"
	"monorepo/bin-webchat-manager/pkg/cachehandler"
)

// DBHandler interface
type DBHandler interface {
	// Widget operations
	WidgetCreate(ctx context.Context, w *widget.Widget) error
	WidgetGet(ctx context.Context, id uuid.UUID) (*widget.Widget, error)
	WidgetList(ctx context.Context, size uint64, token string, filters map[widget.Field]any) ([]*widget.Widget, error)
	WidgetUpdate(ctx context.Context, id uuid.UUID, fields map[widget.Field]any) error
	WidgetDelete(ctx context.Context, id uuid.UUID) error

	// Session operations
	SessionCreate(ctx context.Context, s *session.Session) error
	SessionGet(ctx context.Context, id uuid.UUID) (*session.Session, error)
	SessionList(ctx context.Context, size uint64, token string, filters map[session.Field]any) ([]*session.Session, error)
	SessionUpdate(ctx context.Context, id uuid.UUID, fields map[session.Field]any) error
	SessionDelete(ctx context.Context, id uuid.UUID) error

	// Message operations
	MessageCreate(ctx context.Context, m *message.Message) error
	MessageGet(ctx context.Context, id uuid.UUID) (*message.Message, error)
	MessageList(ctx context.Context, size uint64, token string, filters map[message.Field]any) ([]*message.Message, error)
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

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
	return h
}

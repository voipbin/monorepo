package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"fmt"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-tag-manager/models/tag"
	"monorepo/bin-tag-manager/pkg/cachehandler"
)

// DBHandler interface
type DBHandler interface {
	TagCreate(ctx context.Context, t *tag.Tag) error
	TagDelete(ctx context.Context, id uuid.UUID) error
	TagSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error
	TagGet(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
	TagList(ctx context.Context, size uint64, token string, filters map[tag.Field]any) ([]*tag.Tag, error)
	TagUpdate(ctx context.Context, id uuid.UUID, fields map[tag.Field]any) error
}

// handler database handler
type handler struct {
	utilHandler utilhandler.UtilHandler
	db          *sql.DB
	cache       cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = fmt.Errorf("record not found")
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

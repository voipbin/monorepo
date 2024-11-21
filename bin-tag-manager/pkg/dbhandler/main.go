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
	TagCreate(ctx context.Context, a *tag.Tag) error
	TagDelete(ctx context.Context, id uuid.UUID) error
	TagSetBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error
	TagGet(ctx context.Context, id uuid.UUID) (*tag.Tag, error)
	TagGets(ctx context.Context, customerID uuid.UUID, size uint64, token string) ([]*tag.Tag, error)
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

// List of default values
const (
	DefaultTimeStamp = "9999-01-01 00:00:00.000000" // default timestamp
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

package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-route-manager/models/provider"
	"monorepo/bin-route-manager/models/route"
	"monorepo/bin-route-manager/pkg/cachehandler"
)

// DBHandler interface for route_manager database handle
type DBHandler interface {
	// provider
	ProviderCreate(ctx context.Context, c *provider.Provider) error
	ProviderGet(ctx context.Context, id uuid.UUID) (*provider.Provider, error)
	ProviderGets(ctx context.Context, token string, limit uint64, filters map[provider.Field]any) ([]*provider.Provider, error)
	ProviderDelete(ctx context.Context, id uuid.UUID) error
	ProviderUpdate(ctx context.Context, id uuid.UUID, fields map[provider.Field]any) error

	// route
	RouteCreate(ctx context.Context, r *route.Route) error
	RouteGet(ctx context.Context, id uuid.UUID) (*route.Route, error)
	RouteGets(ctx context.Context, token string, limit uint64, filters map[route.Field]any) ([]*route.Route, error)
	RouteDelete(ctx context.Context, id uuid.UUID) error
	RouteUpdate(ctx context.Context, id uuid.UUID, fields map[route.Field]any) error
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

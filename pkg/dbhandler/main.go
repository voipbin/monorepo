package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/route-manager.git/models/provider"
	"gitlab.com/voipbin/bin-manager/route-manager.git/models/route"
	"gitlab.com/voipbin/bin-manager/route-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	// provider
	ProviderCreate(ctx context.Context, c *provider.Provider) error
	ProviderGet(ctx context.Context, id uuid.UUID) (*provider.Provider, error)
	ProviderGets(ctx context.Context, token string, limit uint64) ([]*provider.Provider, error)
	ProviderDelete(ctx context.Context, id uuid.UUID) error
	ProviderUpdate(ctx context.Context, p *provider.Provider) error

	// route
	RouteCreate(ctx context.Context, r *route.Route) error
	RouteGet(ctx context.Context, id uuid.UUID) (*route.Route, error)
	RouteGets(ctx context.Context, token string, limit uint64) ([]*route.Route, error)
	RouteGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*route.Route, error)
	RouteGetsByCustomerIDWithTarget(ctx context.Context, customerID uuid.UUID, target string) ([]*route.Route, error)
	RouteDelete(ctx context.Context, id uuid.UUID) error
	RouteUpdate(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		providerID uuid.UUID,
		priority int,
		target string,
	) error
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

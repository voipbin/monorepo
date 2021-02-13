package dbhandler

//go:generate mockgen -destination ./mock_dbhandler_dbhandler.go -package dbhandler -source ./main.go DBHandler

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/asterisk"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	// Domains
	// DomainCreate(ctx context.Context)

	// AstAOR
	AstAORCreate(ctx context.Context, b *asterisk.AstAOR) error
	AstAORDelete(ctx context.Context, id string) error
	AstAORGet(ctx context.Context, id string) (*asterisk.AstAOR, error)
	AstAORGetFromCache(ctx context.Context, id string) (*asterisk.AstAOR, error)
	AstAORGetFromDB(ctx context.Context, id string) (*asterisk.AstAOR, error)
	AstAORSetToCache(ctx context.Context, aor *asterisk.AstAOR) error
	AstAORUpdateToCache(ctx context.Context, id string) error

	// AstAuth
	AstAuthCreate(ctx context.Context, b *asterisk.AstAuth) error
	AstAuthDelete(ctx context.Context, id string) error
	AstAuthGet(ctx context.Context, id string) (*asterisk.AstAuth, error)
	AstAuthGetFromCache(ctx context.Context, id string) (*asterisk.AstAuth, error)
	AstAuthGetFromDB(ctx context.Context, id string) (*asterisk.AstAuth, error)
	AstAuthSetToCache(ctx context.Context, auth *asterisk.AstAuth) error
	AstAuthUpdateToCache(ctx context.Context, id string) error

	// AstEndpoint
	AstEndpointCreate(ctx context.Context, b *asterisk.AstEndpoint) error
	AstEndpointDelete(ctx context.Context, id string) error
	AstEndpointGet(ctx context.Context, id string) (*asterisk.AstEndpoint, error)
	AstEndpointGetFromCache(ctx context.Context, id string) (*asterisk.AstEndpoint, error)
	AstEndpointGetFromDB(ctx context.Context, id string) (*asterisk.AstEndpoint, error)
	AstEndpointSetToCache(ctx context.Context, ednpoint *asterisk.AstEndpoint) error
	AstEndpointUpdateToCache(ctx context.Context, id string) error

	// Domain
	DomainCreate(ctx context.Context, b *models.Domain) error
	DomainDelete(ctx context.Context, id uuid.UUID) error
	DomainGet(ctx context.Context, id uuid.UUID) (*models.Domain, error)
	DomainGetFromCache(ctx context.Context, id uuid.UUID) (*models.Domain, error)
	DomainGetFromDB(ctx context.Context, id uuid.UUID) (*models.Domain, error)
	DomainSetToCache(ctx context.Context, e *models.Domain) error
	DomainUpdateToCache(ctx context.Context, id uuid.UUID) error
}

// handler database handler
type handler struct {
	db    *sql.DB
	cache cachehandler.CacheHandler
}

// handler errors
var (
	ErrNotFound = errors.New("Record not found")
)

const defaultDelayTimeout = time.Millisecond * 150

// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		db:    db,
		cache: cache,
	}
	return h
}

// getCurTime return current utc time string
func getCurTime() string {
	now := time.Now().UTC().String()
	res := strings.TrimSuffix(now, " +0000 UTC")

	return res
}

func getStringPointer(v string) *string {
	return &v
}

func getIntegerPointer(v int) *int {
	return &v
}

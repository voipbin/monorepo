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
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	// Domains
	// DomainCreate(ctx context.Context)

	// AstAOR
	AstAORCreate(ctx context.Context, b *models.AstAOR) error
	AstAORDelete(ctx context.Context, id string) error
	AstAORGet(ctx context.Context, id string) (*models.AstAOR, error)
	AstAORGetFromCache(ctx context.Context, id string) (*models.AstAOR, error)
	AstAORGetFromDB(ctx context.Context, id string) (*models.AstAOR, error)
	AstAORSetToCache(ctx context.Context, aor *models.AstAOR) error
	AstAORUpdateToCache(ctx context.Context, id string) error

	// AstAuth
	AstAuthCreate(ctx context.Context, b *models.AstAuth) error
	AstAuthDelete(ctx context.Context, id string) error
	AstAuthDeleteFromCache(ctx context.Context, id string) error
	AstAuthGet(ctx context.Context, id string) (*models.AstAuth, error)
	AstAuthGetFromCache(ctx context.Context, id string) (*models.AstAuth, error)
	AstAuthGetFromDB(ctx context.Context, id string) (*models.AstAuth, error)
	AstAuthSetToCache(ctx context.Context, auth *models.AstAuth) error
	AstAuthUpdate(ctx context.Context, auth *models.AstAuth) error
	AstAuthUpdateToCache(ctx context.Context, id string) error

	// AstContact
	AstContactsSetToCache(ctx context.Context, ednpoint string, contacts []*models.AstContact) error
	AstContactsGetFromCache(ctx context.Context, endpoint string) ([]*models.AstContact, error)
	AstContactGetsByEndpoint(ctx context.Context, endpoint string) ([]*models.AstContact, error)

	// AstEndpoint
	AstEndpointCreate(ctx context.Context, b *models.AstEndpoint) error
	AstEndpointDelete(ctx context.Context, id string) error
	AstEndpointGet(ctx context.Context, id string) (*models.AstEndpoint, error)
	AstEndpointGetFromCache(ctx context.Context, id string) (*models.AstEndpoint, error)
	AstEndpointGetFromDB(ctx context.Context, id string) (*models.AstEndpoint, error)
	AstEndpointSetToCache(ctx context.Context, ednpoint *models.AstEndpoint) error
	AstEndpointUpdateToCache(ctx context.Context, id string) error

	// Domain
	DomainCreate(ctx context.Context, b *models.Domain) error
	DomainDelete(ctx context.Context, id uuid.UUID) error
	DomainDeleteFromCache(ctx context.Context, id uuid.UUID) error
	DomainGet(ctx context.Context, id uuid.UUID) (*models.Domain, error)
	DomainGetByDomainName(ctx context.Context, domainName string) (*models.Domain, error)
	DomainGetFromCache(ctx context.Context, id uuid.UUID) (*models.Domain, error)
	DomainGetFromDB(ctx context.Context, id uuid.UUID) (*models.Domain, error)
	DomainGetsByUserID(ctx context.Context, userID uint64, token string, limit uint64) ([]*models.Domain, error)
	DomainSetToCache(ctx context.Context, e *models.Domain) error
	DomainUpdate(ctx context.Context, b *models.Domain) error
	DomainUpdateToCache(ctx context.Context, id uuid.UUID) error

	// Extension
	ExtensionCreate(ctx context.Context, b *models.Extension) error
	ExtensionDelete(ctx context.Context, id uuid.UUID) error
	ExtensionGet(ctx context.Context, id uuid.UUID) (*models.Extension, error)
	ExtensionGetFromCache(ctx context.Context, id uuid.UUID) (*models.Extension, error)
	ExtensionGetFromDB(ctx context.Context, id uuid.UUID) (*models.Extension, error)
	ExtensionGetsByDomainID(ctx context.Context, domainID uuid.UUID, token string, limit uint64) ([]*models.Extension, error)
	ExtensionSetToCache(ctx context.Context, e *models.Extension) error
	ExtensionUpdate(ctx context.Context, b *models.Extension) error
	ExtensionUpdateToCache(ctx context.Context, id uuid.UUID) error
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

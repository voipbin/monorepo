package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_dbhandler.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astaor"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astendpoint"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/pkg/cachehandler"
)

// list of const variables
const (
	DefaultTimeStamp = "9999-01-01 00:00:00.000000" // default timestamp
)

// DBHandler interface for call_manager database handle
type DBHandler interface {
	// AstAOR
	AstAORCreate(ctx context.Context, b *astaor.AstAOR) error
	AstAORDelete(ctx context.Context, id string) error
	AstAORGet(ctx context.Context, id string) (*astaor.AstAOR, error)

	// AstAuth
	AstAuthCreate(ctx context.Context, b *astauth.AstAuth) error
	AstAuthDelete(ctx context.Context, id string) error
	AstAuthGet(ctx context.Context, id string) (*astauth.AstAuth, error)
	AstAuthUpdate(ctx context.Context, auth *astauth.AstAuth) error

	// AstContact
	AstContactDeleteFromCache(ctx context.Context, endpoint string) error
	AstContactGetsByEndpoint(ctx context.Context, endpoint string) ([]*astcontact.AstContact, error)
	AstContactGetsFromCache(ctx context.Context, endpoint string) ([]*astcontact.AstContact, error)
	AstContactsSetToCache(ctx context.Context, ednpoint string, contacts []*astcontact.AstContact) error

	// AstEndpoint
	AstEndpointCreate(ctx context.Context, b *astendpoint.AstEndpoint) error
	AstEndpointDelete(ctx context.Context, id string) error
	AstEndpointGet(ctx context.Context, id string) (*astendpoint.AstEndpoint, error)

	// Domain
	DomainCreate(ctx context.Context, b *domain.Domain) error
	DomainDelete(ctx context.Context, id uuid.UUID) error
	DomainGet(ctx context.Context, id uuid.UUID) (*domain.Domain, error)
	DomainGetByDomainName(ctx context.Context, domainName string) (*domain.Domain, error)
	DomainGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*domain.Domain, error)
	DomainUpdateBasicInfo(ctx context.Context, id uuid.UUID, name, detail string) error

	// Extension
	ExtensionCreate(ctx context.Context, b *extension.Extension) error
	ExtensionDelete(ctx context.Context, id uuid.UUID) error
	ExtensionGet(ctx context.Context, id uuid.UUID) (*extension.Extension, error)
	ExtensionGetsByDomainID(ctx context.Context, domainID uuid.UUID, token string, limit uint64) ([]*extension.Extension, error)
	ExtensionUpdate(ctx context.Context, b *extension.Extension) error
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

// GetCurTime return current utc time string
func GetCurTime() string {
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

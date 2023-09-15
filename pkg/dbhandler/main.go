package dbhandler

//go:generate go run -mod=mod github.com/golang/mock/mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"github.com/gofrs/uuid"
	"gitlab.com/voipbin/bin-manager/common-handler.git/pkg/utilhandler"

	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astaor"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astauth"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astcontact"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/astendpoint"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/domain"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/extension"
	"gitlab.com/voipbin/bin-manager/registrar-manager.git/models/trunk"
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
	ExtensionGetByEndpointID(ctx context.Context, endpoint string) (*extension.Extension, error)
	ExtensionGetsByDomainID(ctx context.Context, domainID uuid.UUID, token string, limit uint64) ([]*extension.Extension, error)
	ExtensionGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*extension.Extension, error)
	ExtensionUpdate(ctx context.Context, b *extension.Extension) error

	// Trunk
	TrunkCreate(ctx context.Context, t *trunk.Trunk) error
	TrunkUpdateBasicInfo(
		ctx context.Context,
		id uuid.UUID,
		name string,
		detail string,
		authTypes []trunk.AuthType,
		username string,
		password string,
		allowedIPs []string,
	) error
	TrunkGet(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error)
	TrunkGetByDomainName(ctx context.Context, domainName string) (*trunk.Trunk, error)
	TrunkGetsByCustomerID(ctx context.Context, customerID uuid.UUID, token string, limit uint64) ([]*trunk.Trunk, error)
	TrunkDelete(ctx context.Context, id uuid.UUID) error
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

func getStringPointer(v string) *string {
	return &v
}

func getIntegerPointer(v int) *int {
	return &v
}

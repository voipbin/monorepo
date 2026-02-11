package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"errors"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-registrar-manager/models/astaor"
	"monorepo/bin-registrar-manager/models/astauth"
	"monorepo/bin-registrar-manager/models/astcontact"
	"monorepo/bin-registrar-manager/models/astendpoint"
	"monorepo/bin-registrar-manager/models/extension"
	"monorepo/bin-registrar-manager/models/extensiondirect"
	"monorepo/bin-registrar-manager/models/sipauth"
	"monorepo/bin-registrar-manager/models/trunk"
	"monorepo/bin-registrar-manager/pkg/cachehandler"
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

	// Extension
	ExtensionCountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error)
	ExtensionCreate(ctx context.Context, b *extension.Extension) error
	ExtensionDelete(ctx context.Context, id uuid.UUID) error
	ExtensionGet(ctx context.Context, id uuid.UUID) (*extension.Extension, error)
	ExtensionList(ctx context.Context, size uint64, token string, filters map[extension.Field]any) ([]*extension.Extension, error)
	ExtensionGetByEndpointID(ctx context.Context, endpoint string) (*extension.Extension, error)
	ExtensionGetByExtension(ctx context.Context, customerID uuid.UUID, ext string) (*extension.Extension, error)
	ExtensionUpdate(ctx context.Context, id uuid.UUID, fields map[extension.Field]any) error

	// ExtensionDirect
	ExtensionDirectCreate(ctx context.Context, ed *extensiondirect.ExtensionDirect) error
	ExtensionDirectDelete(ctx context.Context, id uuid.UUID) error
	ExtensionDirectGet(ctx context.Context, id uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	ExtensionDirectGetByExtensionID(ctx context.Context, extensionID uuid.UUID) (*extensiondirect.ExtensionDirect, error)
	ExtensionDirectGetByExtensionIDs(ctx context.Context, extensionIDs []uuid.UUID) ([]*extensiondirect.ExtensionDirect, error)
	ExtensionDirectGetByHash(ctx context.Context, hash string) (*extensiondirect.ExtensionDirect, error)
	ExtensionDirectUpdate(ctx context.Context, id uuid.UUID, fields map[extensiondirect.Field]any) error

	// SIPAuth
	SIPAuthCreate(ctx context.Context, t *sipauth.SIPAuth) error
	SIPAuthUpdate(ctx context.Context, id uuid.UUID, fields map[sipauth.Field]any) error
	SIPAuthDelete(ctx context.Context, id uuid.UUID) error
	SIPAuthGet(ctx context.Context, id uuid.UUID) (*sipauth.SIPAuth, error)

	// Trunk
	TrunkCountByCustomerID(ctx context.Context, customerID uuid.UUID) (int, error)
	TrunkCreate(ctx context.Context, t *trunk.Trunk) error
	TrunkUpdate(ctx context.Context, id uuid.UUID, fields map[trunk.Field]any) error
	TrunkGet(ctx context.Context, id uuid.UUID) (*trunk.Trunk, error)
	TrunkGetByDomainName(ctx context.Context, domainName string) (*trunk.Trunk, error)
	TrunkList(ctx context.Context, size uint64, token string, filters map[trunk.Field]any) ([]*trunk.Trunk, error)
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

package customerdomainhandler

//go:generate mockgen -package customerdomainhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	cucustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"

	"monorepo/bin-registrar-manager/models/customerdomain"
	"monorepo/bin-registrar-manager/pkg/dbhandler"
)

// CustomerDomainHandler is interface for customer domain handle
type CustomerDomainHandler interface {
	// EnsureByCustomerID is an idempotent get-or-create: returns the existing
	// customer domain row, or creates one. The shape of a NEW row depends on
	// the short-label flag (VOIP-1385 deploy gate): when short labels are
	// DISABLED (pre-cutover default) the row is created in legacy form
	// (label = customer uuid string, realm = <uuid>.<base>) so frontends that
	// still construct the domain from the customer uuid keep working; when
	// ENABLED it gets a fresh 4-char base36 label. Lazy-create semantics:
	// call this ONLY from the extension-create path (and the customer_created
	// event); read paths must use GetByCustomerID.
	EnsureByCustomerID(ctx context.Context, customerID uuid.UUID) (*customerdomain.CustomerDomain, error)

	// GetByCustomerID is a lookup-only read (no side effects).
	GetByCustomerID(ctx context.Context, customerID uuid.UUID) (*customerdomain.CustomerDomain, error)

	// GetByRealm is a cache-backed lookup-only read (no side effects).
	GetByRealm(ctx context.Context, realm string) (*customerdomain.CustomerDomain, error)

	// DeleteByCustomerID hard-deletes the customer's domain row (customer_deleted cascade).
	DeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error

	// RealmGet returns the customer's realm, lazily creating the domain row
	// when missing (extension-create path only).
	RealmGet(ctx context.Context, customerID uuid.UUID) (string, error)

	// EndpointGet returns the endpoint string (<extension>@<realm>) of the
	// given customer and extension. Lookup-only: it never creates a domain
	// row; a missing row is an error (post-cutover every active customer has
	// a domain row).
	EndpointGet(ctx context.Context, customerID uuid.UUID, extension string) (string, error)

	// MigrateToShortDomain moves the customer's domain row to a fresh 4-char
	// label (migration batch): creates the row when missing, rewrites a
	// backfilled long-label row in place, and no-ops when the row already
	// carries a short label (idempotent resume). ALWAYS generates short
	// labels regardless of the short-label flag — the batch IS the cutover
	// mechanism.
	MigrateToShortDomain(ctx context.Context, customerID uuid.UUID) (*customerdomain.CustomerDomain, error)

	// UpdateByCustomerID sets the customer's domain row to the given explicit
	// label/realm (migration rollback replay).
	UpdateByCustomerID(ctx context.Context, customerID uuid.UUID, domainLabel string, realm string) (*customerdomain.CustomerDomain, error)

	EventCUCustomerCreated(ctx context.Context, cu *cucustomer.Customer) error
	EventCUCustomerDeleted(ctx context.Context, cu *cucustomer.Customer) error
}

// customerDomainHandler structure for customer domain handle
type customerDomainHandler struct {
	dbBin dbhandler.DBHandler

	// shortLabelEnabled gates short 4-char label generation for NEW rows
	// (config domain_short_label_enabled / env DOMAIN_SHORT_LABEL_ENABLED).
	// false (pre-cutover default): EnsureByCustomerID creates legacy uuid
	// label/realm rows. The migration batch is NOT gated by this flag.
	shortLabelEnabled bool
}

// NewCustomerDomainHandler returns new customer domain handler
func NewCustomerDomainHandler(dbBin dbhandler.DBHandler, shortLabelEnabled bool) CustomerDomainHandler {
	h := &customerDomainHandler{
		dbBin:             dbBin,
		shortLabelEnabled: shortLabelEnabled,
	}

	return h
}

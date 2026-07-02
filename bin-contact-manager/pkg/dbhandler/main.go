package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/interaction"
	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/cachehandler"
)

// DBHandler interface for contact-manager database operations
type DBHandler interface {
	// Contact operations
	ContactCreate(ctx context.Context, c *contact.Contact) error
	ContactGet(ctx context.Context, id uuid.UUID) (*contact.Contact, error)
	ContactList(ctx context.Context, size uint64, token string, filters map[contact.Field]any) ([]*contact.Contact, error)
	ContactUpdate(ctx context.Context, id uuid.UUID, fields map[contact.Field]any) error
	ContactDelete(ctx context.Context, id uuid.UUID) error
	ContactLookupByPhone(ctx context.Context, customerID uuid.UUID, phoneE164 string) (*contact.Contact, error)
	ContactLookupByEmail(ctx context.Context, customerID uuid.UUID, email string) (*contact.Contact, error)
	ContactDeleteByCustomerID(ctx context.Context, customerID uuid.UUID) error

	// Address operations
	AddressCreate(ctx context.Context, a *contact.Address) error
	AddressGet(ctx context.Context, customerID, id uuid.UUID) (*contact.Address, error)
	AddressList(ctx context.Context, customerID uuid.UUID, filters map[string]any, pageToken string, pageSize uint64) ([]contact.Address, error)
	AddressListByContactID(ctx context.Context, contactID uuid.UUID) ([]contact.Address, error)
	AddressUpdate(ctx context.Context, id uuid.UUID, fields map[string]any) error
	AddressDelete(ctx context.Context, id uuid.UUID) error
	AddressResetPrimary(ctx context.Context, contactID uuid.UUID) error
	AddressClaim(ctx context.Context, customerID, addressID, contactID uuid.UUID) error

	// TagAssignment operations
	TagAssignmentCreate(ctx context.Context, contactID, tagID uuid.UUID) error
	TagAssignmentDelete(ctx context.Context, contactID, tagID uuid.UUID) error
	TagAssignmentListByContactID(ctx context.Context, contactID uuid.UUID) ([]uuid.UUID, error)

	// Interaction operations
	InteractionCreate(ctx context.Context, i *interaction.Interaction) error
	InteractionGet(ctx context.Context, id uuid.UUID) (*interaction.Interaction, error)
	InteractionList(ctx context.Context, customerID uuid.UUID, size uint64, token string, peerType, peerTarget string, addressSet []AddressPair) ([]*interaction.Interaction, error)
	InteractionListByIDs(ctx context.Context, customerID uuid.UUID, ids []uuid.UUID) ([]*interaction.Interaction, error)
	InteractionListUnresolved(ctx context.Context, customerID uuid.UUID, size uint64, token string, since time.Time) ([]*interaction.Interaction, error)

	// Resolution operations
	ResolutionCreate(ctx context.Context, r *resolution.Resolution) error
	ResolutionDelete(ctx context.Context, customerID, interactionID, id uuid.UUID) error
	ResolutionListByInteraction(ctx context.Context, customerID, interactionID uuid.UUID) ([]*resolution.Resolution, error)
	ResolutionListByContact(ctx context.Context, customerID, contactID uuid.UUID) ([]*resolution.Resolution, error)
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
	ErrConflict = fmt.Errorf("address already claimed")

	// ErrDuplicateTarget is returned by AddressCreate when the insert
	// violates the unique index on contact_addresses(customer_id, type,
	// target). Distinct from ErrConflict, whose message ("address already
	// claimed") is specific to the ClaimAddress flow.
	ErrDuplicateTarget = fmt.Errorf("address already exists for this customer")
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

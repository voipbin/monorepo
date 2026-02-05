package dbhandler

//go:generate mockgen -package dbhandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"database/sql"
	"fmt"

	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/contact"
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

	// PhoneNumber operations
	PhoneNumberCreate(ctx context.Context, p *contact.PhoneNumber) error
	PhoneNumberDelete(ctx context.Context, id uuid.UUID) error
	PhoneNumberListByContactID(ctx context.Context, contactID uuid.UUID) ([]contact.PhoneNumber, error)

	// Email operations
	EmailCreate(ctx context.Context, e *contact.Email) error
	EmailDelete(ctx context.Context, id uuid.UUID) error
	EmailListByContactID(ctx context.Context, contactID uuid.UUID) ([]contact.Email, error)

	// TagAssignment operations
	TagAssignmentCreate(ctx context.Context, contactID, tagID uuid.UUID) error
	TagAssignmentDelete(ctx context.Context, contactID, tagID uuid.UUID) error
	TagAssignmentListByContactID(ctx context.Context, contactID uuid.UUID) ([]uuid.UUID, error)
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


// NewHandler creates DBHandler
func NewHandler(db *sql.DB, cache cachehandler.CacheHandler) DBHandler {
	h := &handler{
		utilHandler: utilhandler.NewUtilHandler(),
		db:          db,
		cache:       cache,
	}
	return h
}

package contacthandler

//go:generate mockgen -package contacthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/dbhandler"
)

// ContactHandler interface for contact business logic operations
type ContactHandler interface {
	// Contact operations
	Create(ctx context.Context, c *contact.Contact) (*contact.Contact, error)
	Get(ctx context.Context, id uuid.UUID) (*contact.Contact, error)
	List(ctx context.Context, size uint64, token string, filters map[contact.Field]any) ([]*contact.Contact, error)
	Update(ctx context.Context, id uuid.UUID, fields map[contact.Field]any) (*contact.Contact, error)
	Delete(ctx context.Context, id uuid.UUID) (*contact.Contact, error)
	LookupByPhone(ctx context.Context, customerID uuid.UUID, phoneE164 string) (*contact.Contact, error)
	LookupByEmail(ctx context.Context, customerID uuid.UUID, email string) (*contact.Contact, error)

	// PhoneNumber operations
	AddPhoneNumber(ctx context.Context, contactID uuid.UUID, p *contact.PhoneNumber) (*contact.Contact, error)
	UpdatePhoneNumber(ctx context.Context, contactID, phoneID uuid.UUID, fields map[string]any) (*contact.Contact, error)
	RemovePhoneNumber(ctx context.Context, contactID, phoneID uuid.UUID) (*contact.Contact, error)

	// Email operations
	AddEmail(ctx context.Context, contactID uuid.UUID, e *contact.Email) (*contact.Contact, error)
	UpdateEmail(ctx context.Context, contactID, emailID uuid.UUID, fields map[string]any) (*contact.Contact, error)
	RemoveEmail(ctx context.Context, contactID, emailID uuid.UUID) (*contact.Contact, error)

	// Tag operations
	AddTag(ctx context.Context, contactID, tagID uuid.UUID) (*contact.Contact, error)
	RemoveTag(ctx context.Context, contactID, tagID uuid.UUID) (*contact.Contact, error)

	// Event handlers
	EventCustomerDeleted(ctx context.Context, c *cmcustomer.Customer) error
}

type contactHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
}

// NewContactHandler returns ContactHandler interface
func NewContactHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
) ContactHandler {
	return &contactHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyHandler: notifyHandler,
	}
}

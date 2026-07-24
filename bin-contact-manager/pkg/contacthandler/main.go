package contacthandler

//go:generate mockgen -package contacthandler -destination ./mock_main.go -source main.go -build_flags=-mod=mod

import (
	"context"
	"time"

	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"

	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
	tmpeerevent "monorepo/bin-timeline-manager/models/peerevent"
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

	// Address operations
	AddAddress(ctx context.Context, contactID uuid.UUID, a *contact.Address) (*contact.Address, error)
	UpdateAddress(ctx context.Context, contactID, addressID uuid.UUID, fields map[string]any) (*contact.Contact, error)
	RemoveAddress(ctx context.Context, contactID, addressID uuid.UUID) (*contact.Contact, error)
	CreateUnresolvedAddress(ctx context.Context, customerID uuid.UUID, a *contact.Address) (*contact.Address, error)
	ClaimAddress(ctx context.Context, customerID, addressID, contactID uuid.UUID) (*contact.Address, error)

	// Tag operations
	AddTag(ctx context.Context, contactID, tagID uuid.UUID) (*contact.Contact, error)
	RemoveTag(ctx context.Context, contactID, tagID uuid.UUID) (*contact.Contact, error)

	// Event handlers
	EventCustomerDeleted(ctx context.Context, c *cmcustomer.Customer) error

	// Interaction read operations (CRM v1 read API, VOIP-1209).
	// Proxies bin-timeline-manager's peer_events read API (design doc
	// 2026-07-25-contact-interaction-retire-to-peer-events, §8.1/§9).
	InteractionList(ctx context.Context, customerID uuid.UUID, size uint64, token string,
		peerType, peerTarget string, contactID uuid.UUID, addressID uuid.UUID, since time.Time) ([]*tmpeerevent.PeerEvent, string, error)
}

type contactHandler struct {
	utilHandler   utilhandler.UtilHandler
	reqHandler    requesthandler.RequestHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
	caseHandler   casehandler.CaseHandler
}

// NewContactHandler returns ContactHandler interface
func NewContactHandler(
	reqHandler requesthandler.RequestHandler,
	dbHandler dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	caseHandler casehandler.CaseHandler,
) ContactHandler {
	return &contactHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		reqHandler:    reqHandler,
		db:            dbHandler,
		notifyHandler: notifyHandler,
		caseHandler:   caseHandler,
	}
}

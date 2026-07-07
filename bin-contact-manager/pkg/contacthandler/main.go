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

	callmodel "monorepo/bin-call-manager/models/call"
	"monorepo/bin-contact-manager/models/contact"
	"monorepo/bin-contact-manager/models/interaction"
	"monorepo/bin-contact-manager/models/resolution"
	"monorepo/bin-contact-manager/pkg/casehandler"
	"monorepo/bin-contact-manager/pkg/dbhandler"
	convmsg "monorepo/bin-conversation-manager/models/message"
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

	// Projection event handlers (CRM interaction timeline)
	EventCallCreated(ctx context.Context, m *callmodel.WebhookMessage) error
	EventConversationMessageCreated(ctx context.Context, m *convmsg.WebhookMessage) error

	// Interaction read operations (CRM v1 read API, VOIP-1209)
	InteractionGet(ctx context.Context, customerID, id uuid.UUID) (*interaction.Interaction, error)
	InteractionList(ctx context.Context, customerID uuid.UUID, size uint64, token string,
		peerType, peerTarget string, contactID uuid.UUID, addressID uuid.UUID, since time.Time) ([]*interaction.Interaction, string, error)
	InteractionListUnresolved(ctx context.Context, customerID uuid.UUID, size uint64, token string, since time.Time) ([]*interaction.Interaction, string, error)

	// Resolution operations (CRM v1 attribution, VOIP-1209)
	ResolutionCreate(ctx context.Context, customerID, contactID, interactionID uuid.UUID,
		resolutionType, resolvedByType string, resolvedByID uuid.UUID) (*resolution.Resolution, error)
	ResolutionDelete(ctx context.Context, customerID, interactionID, id uuid.UUID) error
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

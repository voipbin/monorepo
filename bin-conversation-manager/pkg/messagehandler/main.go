package messagehandler

//go:generate mockgen -package messagehandler -destination ./mock_messagehandler.go -source main.go -build_flags=-mod=mod

import (
	"context"

	commonaddress "monorepo/bin-common-handler/models/address"
	"monorepo/bin-common-handler/pkg/notifyhandler"
	"monorepo/bin-common-handler/pkg/requesthandler"
	"monorepo/bin-common-handler/pkg/utilhandler"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"
	"monorepo/bin-conversation-manager/pkg/accounthandler"
	"monorepo/bin-conversation-manager/pkg/dbhandler"
	"monorepo/bin-conversation-manager/pkg/linehandler"
	"monorepo/bin-conversation-manager/pkg/smshandler"
	"monorepo/bin-conversation-manager/pkg/whatsapphandler"
)

// MessageHandler defiens
type MessageHandler interface {
	Create(ctx context.Context, args MessageCreateArgs) (*message.Message, error)
	Delete(ctx context.Context, id uuid.UUID) (*message.Message, error)
	Get(ctx context.Context, id uuid.UUID) (*message.Message, error)
	List(ctx context.Context, pageToken string, pageSize uint64, filters map[message.Field]any) ([]*message.Message, error)
	UpdateStatus(ctx context.Context, id uuid.UUID, status message.Status) (*message.Message, error)

	Send(ctx context.Context, cv *conversation.Conversation, text string, medias []media.Media) (*message.Message, error)
}

// MessageCreateArgs holds the inputs for creating a conversation message.
//
// Source/Destination are the absolute endpoints the message carried (source =
// sending party, destination = receiving party). They are caller-derived via
// DeriveEndpoints(cv, direction) at the call site where the conversation is in
// hand; Create only stores them (it receives no conversation, so it never
// re-derives / re-fetches). See the VOIP-1215 design.
type MessageCreateArgs struct {
	ID             uuid.UUID
	CustomerID     uuid.UUID
	ConversationID uuid.UUID
	Direction      message.Direction
	Status         message.Status
	ReferenceType  message.ReferenceType
	ReferenceID    uuid.UUID
	TransactionID  string
	Text           string
	Subject        string
	Medias         []media.Media

	Source      commonaddress.Address
	Destination commonaddress.Address

	// CaseID is the case-linking hint to attach to the created message's
	// event payload (contact-case-management design §4.3), sourced by
	// the caller (message.go's MessageEventReceived/MessageEventSent)
	// from the owning Conversation's Metadata.ContactCaseID. Not
	// persisted to the message row (Message.CaseID is db:"-"); Create
	// re-attaches it to the post-DB-read result before publishing the
	// event, since the DB round-trip would otherwise drop it.
	CaseID *uuid.UUID
}

// DeriveEndpoints maps a conversation's relative Self/Peer to a message's
// absolute source/destination by direction. This is the single authority for the
// VOIP-1215 fill rule; every Create call site uses it so the rule cannot be
// applied inconsistently.
//
//	outgoing (outbound): we are the sender   -> source = Self, destination = Peer
//	incoming (inbound):  remote is the sender -> source = Peer, destination = Self
//	unknown ("" / other): do NOT guess; return zero endpoints (caller logs).
func DeriveEndpoints(cv *conversation.Conversation, dir message.Direction) (source, destination commonaddress.Address) {
	if cv == nil {
		return commonaddress.Address{}, commonaddress.Address{}
	}
	switch dir {
	case message.DirectionOutgoing:
		return cv.Self, cv.Peer
	case message.DirectionIncoming:
		return cv.Peer, cv.Self
	default:
		return commonaddress.Address{}, commonaddress.Address{}
	}
}

type messageHandler struct {
	utilHandler   utilhandler.UtilHandler
	db            dbhandler.DBHandler
	notifyHandler notifyhandler.NotifyHandler
	reqHandler    requesthandler.RequestHandler

	accountHandler  accounthandler.AccountHandler
	lineHandler     linehandler.LineHandler
	smsHandler      smshandler.SMSHandler
	whatsappHandler whatsapphandler.WhatsAppHandler
}

// NewMessageHandler returns a new ConversationHandler
func NewMessageHandler(
	db dbhandler.DBHandler,
	notifyHandler notifyhandler.NotifyHandler,
	reqHandler requesthandler.RequestHandler,
	accountHandler accounthandler.AccountHandler,
	lineHandler linehandler.LineHandler,
	smsHandler smshandler.SMSHandler,
	whatsappHandler whatsapphandler.WhatsAppHandler,
) MessageHandler {
	return &messageHandler{
		utilHandler:   utilhandler.NewUtilHandler(),
		db:            db,
		notifyHandler: notifyHandler,
		reqHandler:    reqHandler,

		accountHandler:  accountHandler,
		lineHandler:     lineHandler,
		smsHandler:      smsHandler,
		whatsappHandler: whatsappHandler,
	}
}

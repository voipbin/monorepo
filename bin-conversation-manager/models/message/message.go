package message

import (
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/media"
)

// Message defines
type Message struct {
	commonidentity.Identity

	ConversationID uuid.UUID `json:"conversation_id,omitempty" db:"conversation_id,uuid"`
	Direction      Direction `json:"direction,omitempty" db:"direction"`
	Status         Status    `json:"status,omitempty" db:"status"`

	ReferenceType ReferenceType `json:"reference_type,omitempty" db:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty" db:"reference_id,uuid"`

	// Absolute endpoints the message carried: source = sending party,
	// destination = receiving party (direction-independent in meaning). Set once
	// at creation from the conversation's Self/Peer + direction (see
	// messagehandler.deriveEndpoints). Stored as JSON, mirroring
	// conversation.Self/Peer.
	Source      commonaddress.Address `json:"source,omitempty" db:"source,json"`
	Destination commonaddress.Address `json:"destination,omitempty" db:"destination,json"`

	TransactionID string `json:"transaction_id,omitempty" db:"transaction_id"` // uniq id for message's transaction

	Text    string        `json:"text,omitempty" db:"text"`
	Subject string        `json:"subject,omitempty" db:"subject"` // email subject; empty for non-email messages
	Medias  []media.Media `json:"medias,omitempty" db:"medias,json"`

	// CaseID is the (single) case-linking hint this event carries,
	// sourced from the owning Conversation's Metadata.ContactCaseID at
	// the moment this message was created (contact-case-management
	// design §4.3). Purely internal plumbing for bin-contact-manager's
	// Case get-or-create -- deliberately NOT copied by
	// ConvertWebhookMessage onto the customer-facing webhook payload
	// (see webhook.go). Not persisted as its own DB column; it is a
	// point-in-time snapshot taken when the event is built, not a
	// stored field of the message row.
	CaseID *uuid.UUID `json:"case_id,omitempty" db:"-"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// Field defines the fields for the Message entity.
type Field string

// List of message fields
const (
	FieldID         Field = "id"
	FieldCustomerID Field = "customer_id"

	FieldConversationID Field = "conversation_id"
	FieldDirection      Field = "direction"
	FieldStatus         Field = "status"

	FieldReferenceType Field = "reference_type"
	FieldReferenceID   Field = "reference_id"

	FieldSource      Field = "source"
	FieldDestination Field = "destination"

	FieldTransactionID Field = "transaction_id"

	FieldText    Field = "text"
	FieldSubject Field = "subject"
	FieldMedias  Field = "medias" // Stored as JSON

	FieldTMCreate Field = "tm_create"
	FieldTMUpdate Field = "tm_update"
	FieldTMDelete Field = "tm_delete"

	// filter only
	FieldDeleted Field = "deleted"
)

// Status defines
type Status string

// list of Status
const (
	StatusFailed      Status = "failed"
	StatusProgressing Status = "progressing"
	StatusDone        Status = "done"
)

// Direction message's direction
type Direction string

// list of Direction defines
const (
	DirectionNond     Direction = ""
	DirectionOutgoing Direction = "outgoing"
	DirectionIncoming Direction = "incoming"
)

type ReferenceType string

const (
	ReferenceTypeNone     ReferenceType = ""
	ReferenceTypeMessage  ReferenceType = "message" // sms, mms
	ReferenceTypeLine     ReferenceType = "line"
	ReferenceTypeWhatsApp ReferenceType = "whatsapp"
	ReferenceTypeEmail    ReferenceType = "email"
	ReferenceTypeWebchat  ReferenceType = "webchat"
)

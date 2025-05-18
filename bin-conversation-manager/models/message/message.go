package message

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/media"
)

// Message defines
type Message struct {
	commonidentity.Identity

	ConversationID uuid.UUID `json:"conversation_id,omitempty"`
	Direction      Direction `json:"direction,omitempty"`
	Status         Status    `json:"status,omitempty"`

	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	TransactionID string `json:"transaction_id,omitempty"` // uniq id for message's transaction

	Text   string        `json:"text,omitempty"`
	Medias []media.Media `json:"medias,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
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

	FieldTransactionID Field = "transaction_id"

	FieldText   Field = "text"
	FieldMedias Field = "medias" // Stored as JSON

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
	ReferenceTypeNone    ReferenceType = ""
	ReferenceTypeMessage ReferenceType = "message" // sms, mms
	ReferenceTypeLine    ReferenceType = "line"
)

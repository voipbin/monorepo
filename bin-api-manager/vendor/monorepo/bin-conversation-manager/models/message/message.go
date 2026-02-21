package message

import (
	"time"

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

	TransactionID string `json:"transaction_id,omitempty" db:"transaction_id"` // uniq id for message's transaction

	Text   string        `json:"text,omitempty" db:"text"`
	Medias []media.Media `json:"medias,omitempty" db:"medias,json"`

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

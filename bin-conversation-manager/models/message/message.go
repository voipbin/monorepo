package message

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
)

// Message defines
type Message struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	ConversationID uuid.UUID `json:"conversation_id"`
	Direction      Direction `json:"direction"`
	Status         Status    `json:"status"`

	ReferenceType conversation.ReferenceType `json:"reference_type"` // used for find a conversation info(source info: group/room/user)
	ReferenceID   string                     `json:"reference_id"`   // used for find a conversation info(source info: group_id, room_id, user_id)

	TransactionID string `json:"transaction_id"` // uniq id for message's transaction

	Source *commonaddress.Address `json:"source"` // source

	Text   string        `json:"text"`
	Medias []media.Media `json:"medias"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Status defines
type Status string

// list of Status
const (
	StatusSending  Status = "sending"
	StatusSent     Status = "sent"
	StatusFailed   Status = "failed"
	StatusReceived Status = "received"
)

// Direction message's direction
type Direction string

// list of Direction defines
const (
	DirectionNond     Direction = ""
	DirectionOutgoing Direction = "outgoing"
	DirectionIncoming Direction = "incoming"
)

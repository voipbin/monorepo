package message

import (
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"

	"monorepo/bin-conversation-manager/models/conversation"
	"monorepo/bin-conversation-manager/models/media"
)

// Message defines
type Message struct {
	commonidentity.Identity

	ConversationID uuid.UUID `json:"conversation_id"`
	Direction      Direction `json:"direction"`
	Status         Status    `json:"status"`

	ReferenceType conversation.ReferenceType `json:"reference_type"`
	ReferenceID   string                     `json:"reference_id"`

	TransactionID string `json:"transaction_id"` // uniq id for message's transaction

	// Source      *commonaddress.Address `json:"source"`      // source
	// Destination *commonaddress.Address `json:"destination"` // destination

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
	StatusSending     Status = "sending"
	StatusSent        Status = "sent"
	StatusFailed      Status = "failed"
	StatusReceived    Status = "received"
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

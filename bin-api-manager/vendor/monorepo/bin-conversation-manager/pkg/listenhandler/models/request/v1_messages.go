package request

import (
	"monorepo/bin-conversation-manager/models/media"
	"monorepo/bin-conversation-manager/models/message"

	"github.com/gofrs/uuid"
)

// V1DataMessagesPost is
// v1 data type request struct for
// /v1/messages POST
type V1DataMessagesPost struct {
	ConversationID uuid.UUID     `json:"conversation_id,omitempty"`
	Text           string        `json:"text,omitempty"`
	Medias         []media.Media `json:"medias,omitempty"`
}

// V1DataMessagesCreatePost is
// v1 data type request struct for
// /v1/messages/create POST
type V1DataMessagesCreatePost struct {
	ID             uuid.UUID             `json:"id,omitempty"`
	CustomerID     uuid.UUID             `json:"customer_id,omitempty"`
	ConversationID uuid.UUID             `json:"conversation_id,omitempty"`
	Direction      message.Direction     `json:"direction,omitempty"`
	Status         message.Status        `json:"status,omitempty"`
	ReferenceType  message.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID    uuid.UUID             `json:"reference_id,omitempty"`
	TransactionID  string                `json:"transaction_id,omitempty"` // uniq id for message's transaction
	Text           string                `json:"text,omitempty"`
	Medias         []media.Media         `json:"medias,omitempty"`
}

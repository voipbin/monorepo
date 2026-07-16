package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-webchat-manager/models/message"
)

// V1DataMessagesPost is
// v1 data type request struct for
// /v1/messages POST
type V1DataMessagesPost struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	SessionID  uuid.UUID `json:"session_id,omitempty"`

	Direction message.Direction `json:"direction,omitempty"`
	SenderID  uuid.UUID         `json:"sender_id,omitempty"`

	Text string `json:"text,omitempty"`
}

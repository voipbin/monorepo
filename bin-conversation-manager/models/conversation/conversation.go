package conversation

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// Conversation defines
type Conversation struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`
	AccountID  uuid.UUID `json:"account_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   string        `json:"reference_id"`

	Source       *commonaddress.Address  `json:"source"`
	Participants []commonaddress.Address `json:"participants"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ReferenceType defines
type ReferenceType string

// list of reference types
const (
	ReferenceTypeNone                  = ""
	ReferenceTypeMessage ReferenceType = "message" // sms, mms
	ReferenceTypeLine    ReferenceType = "line"
)

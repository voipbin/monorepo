package conversation

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/conversation-manager.git/models/participant"
)

// Conversation defines
type Conversation struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   string        `json:"reference_id"`

	Participants []participant.Participant `json:"participants"`

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

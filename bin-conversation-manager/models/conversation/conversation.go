package conversation

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Conversation represents a conversation entity with associated metadata and details.
// It includes information about the account, reference type, and reference ID,
// as well as addresses for the self and peer participants. Additionally, it tracks
// timestamps for creation, update, and deletion events.
type Conversation struct {
	commonidentity.Identity
	commonidentity.Owner

	AccountID uuid.UUID `json:"account_id,omitempty"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	ReferenceType ReferenceType `json:"reference_type,omitempty"` // represent the type of referenced conversation type
	ReferenceID   string        `json:"reference_id,omitempty"`   // represent the id of referenced conversation transaction. for the Line could be chatroom id, for sms/mms, could be empty.

	Self *commonaddress.Address `json:"self,omitempty"` // self address
	Peer *commonaddress.Address `json:"peer,omitempty"` // peer address

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// ReferenceType defines
type ReferenceType string

// list of reference types
const (
	ReferenceTypeNone                  = ""
	ReferenceTypeMessage ReferenceType = "message" // sms, mms
	ReferenceTypeLine    ReferenceType = "line"
)

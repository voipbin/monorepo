package transfer

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Transfer defines
type Transfer struct {
	commonidentity.Identity

	Type Type `json:"type"`

	// transferer/transferee info
	TransfererCallID    uuid.UUID               `json:"transferer_call_id"`
	TransfereeAddresses []commonaddress.Address `json:"transferee_addresses"`
	TransfereeCallID    uuid.UUID               `json:"transferee_call_id"`

	GroupcallID  uuid.UUID `json:"groupcall_id"` // created groupcall id
	ConfbridgeID uuid.UUID `json:"confbridge_id"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type define
type Type string

// list of types
const (
	TypeAttended Type = "attended"
	TypeBlind    Type = "blind"
)

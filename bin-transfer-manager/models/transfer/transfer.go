package transfer

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Transfer defines
type Transfer struct {
	commonidentity.Identity

	Type Type `json:"type" db:"type"`

	// transferer/transferee info
	TransfererCallID    uuid.UUID               `json:"transferer_call_id" db:"transferer_call_id,uuid"`
	TransfereeAddresses []commonaddress.Address `json:"transferee_addresses" db:"transferee_addresses,json"`
	TransfereeCallID    uuid.UUID               `json:"transferee_call_id" db:"transferee_call_id,uuid"`

	GroupcallID  uuid.UUID `json:"groupcall_id" db:"groupcall_id,uuid"` // created groupcall id
	ConfbridgeID uuid.UUID `json:"confbridge_id" db:"confbridge_id,uuid"`

	TMCreate string `json:"tm_create" db:"tm_create"`
	TMUpdate string `json:"tm_update" db:"tm_update"`
	TMDelete string `json:"tm_delete" db:"tm_delete"`
}

// Type define
type Type string

// list of types
const (
	TypeAttended Type = "attended"
	TypeBlind    Type = "blind"
)

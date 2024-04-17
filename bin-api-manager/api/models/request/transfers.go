package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	tmtransfer "monorepo/bin-transfer-manager/models/transfer"

	"github.com/gofrs/uuid"
)

// BodyTransfersPOST defines request body for
// POST /v1.0/transfers
type BodyTransfersPOST struct {
	TransferType        tmtransfer.Type         `json:"type,omitempty"`
	TransfererCallID    uuid.UUID               `json:"transferer_call_id,omitempty"`
	TransfereeAddresses []commonaddress.Address `json:"transferee_addresses,omitempty"`
}

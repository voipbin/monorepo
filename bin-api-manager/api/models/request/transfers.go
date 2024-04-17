package request

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	tmtransfer "gitlab.com/voipbin/bin-manager/transfer-manager.git/models/transfer"
)

// BodyTransfersPOST defines request body for
// POST /v1.0/transfers
type BodyTransfersPOST struct {
	TransferType        tmtransfer.Type         `json:"type,omitempty"`
	TransfererCallID    uuid.UUID               `json:"transferer_call_id,omitempty"`
	TransfereeAddresses []commonaddress.Address `json:"transferee_addresses,omitempty"`
}

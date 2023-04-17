package request

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/transfer-manager.git/models/transfer"
)

// V1DataTransfersPost is
// v1 data type request struct for
// /v1/transfers POST
type V1DataTransfersPost struct {
	Type                transfer.Type           `json:"type"`
	TransfererCallID    uuid.UUID               `json:"transferer_call_id"`
	TransfereeAddresses []commonaddress.Address `json:"transferee_addresses"`
}

package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-transfer-manager/models/transfer"
)

// V1DataTransfersPost is
// v1 data type request struct for
// /v1/transfers POST
type V1DataTransfersPost struct {
	Type                transfer.Type           `json:"type"`
	TransfererCallID    uuid.UUID               `json:"transferer_call_id"`
	TransfereeAddresses []commonaddress.Address `json:"transferee_addresses"`
}

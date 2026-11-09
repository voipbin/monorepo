package request

import (
	"github.com/gofrs/uuid"

	commonaddress "monorepo/bin-common-handler/models/address"
)

// V1DataPushRingsPost is
// v1 data type for
// /v1/push-rings POST
type V1DataPushRingsPost struct {
	CustomerID   uuid.UUID             `json:"customer_id,omitempty"`
	Destination  commonaddress.Address `json:"destination,omitempty"`
	ConfbridgeID uuid.UUID             `json:"confbridge_id,omitempty"`
	MasterCallID uuid.UUID             `json:"master_call_id,omitempty"`
	FlowID       uuid.UUID             `json:"flow_id,omitempty"`
	Source       commonaddress.Address `json:"source,omitempty"`
}

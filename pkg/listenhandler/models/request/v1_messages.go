package request

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// V1DataMessagesPost is
// v1 data type request struct for
// /v1/messages POST
type V1DataMessagesPost struct {
	ID           uuid.UUID               `json:"id"`
	CustomerID   uuid.UUID               `json:"customer_id"`
	Source       *commonaddress.Address  `json:"source"`
	Destinations []commonaddress.Address `json:"destinations"`
	Text         string                  `json:"text"`
}

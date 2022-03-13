package request

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
)

// V1DataMessagesPost is
// v1 data type request struct for
// /v1/messages POST
type V1DataMessagesPost struct {
	CustomerID   uuid.UUID           `json:"customer_id"`
	Source       *cmaddress.Address  `json:"source"`
	Destinations []cmaddress.Address `json:"destinations"`
	Text         string              `json:"text"`
}

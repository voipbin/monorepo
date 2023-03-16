package request

import (
	"github.com/gofrs/uuid"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

	"gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
)

// V1DataGroupcallsPost is
// v1 data type request struct for
// /v1/groupcalls POST
type V1DataGroupcallsPost struct {
	CustomerID   uuid.UUID               `json:"customer_id,omitempty"`
	Source       commonaddress.Address   `json:"source,omitempty"`
	Destinations []commonaddress.Address `json:"destinations,omitempty"`
	FlowID       uuid.UUID               `json:"flow_id,omitempty"`
	MasterCallID uuid.UUID               `json:"master_call_id,omitempty"`
	RingMethod   groupcall.RingMethod    `json:"ring_method,omitempty"`
	AnswerMethod groupcall.AnswerMethod  `json:"answer_method,omitempty"`
	Connect      bool                    `json:"connect,omitempty"` //Deprecated: Connect // if the call is created for connect, sets this to true,
}

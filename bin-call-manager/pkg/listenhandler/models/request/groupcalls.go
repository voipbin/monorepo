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
	ID                uuid.UUID               `json:"id,omitempty"`
	CustomerID        uuid.UUID               `json:"customer_id,omitempty"`
	FlowID            uuid.UUID               `json:"flow_id,omitempty"`
	Source            commonaddress.Address   `json:"source,omitempty"`
	Destinations      []commonaddress.Address `json:"destinations,omitempty"`
	MasterCallID      uuid.UUID               `json:"master_call_id,omitempty"`
	MasterGroupcallID uuid.UUID               `json:"master_groupcall_id,omitempty"`
	RingMethod        groupcall.RingMethod    `json:"ring_method,omitempty"`
	AnswerMethod      groupcall.AnswerMethod  `json:"answer_method,omitempty"`
}

// V1DataGroupcallsIDAnswerGroupcallIDPost is
// v1 data type request struct for
// /v1/groupcalls/<groupcall-id>/answer_groupcall_id POST
type V1DataGroupcallsIDAnswerGroupcallIDPost struct {
	AnswerGroupcallID uuid.UUID `json:"answer_groupcall_id,omitempty"`
}

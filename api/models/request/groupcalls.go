package request

import (
	"github.com/gofrs/uuid"
	cmgroupcall "gitlab.com/voipbin/bin-manager/call-manager.git/models/groupcall"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
	fmaction "gitlab.com/voipbin/bin-manager/flow-manager.git/models/action"
)

// BodyGroupcallsPOST is rquest body define for
// POST /v1.0/groupcalls
type BodyGroupcallsPOST struct {
	Source       commonaddress.Address    `json:"source" binding:"required"`
	Destinations []commonaddress.Address  `json:"destinations" binding:"required"`
	FlowID       uuid.UUID                `json:"flow_id"`
	Actions      []fmaction.Action        `json:"actions"`
	RingMethod   cmgroupcall.RingMethod   `json:"ring_method"`
	AnswerMethod cmgroupcall.AnswerMethod `json:"answer_method"`
}

// ParamGroupcallsGET is rquest param define for
// GET /v1.0/groupcalls
type ParamGroupcallsGET struct {
	Pagination
}

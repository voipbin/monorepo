package request

import (
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"

	commonaddress "monorepo/bin-common-handler/models/address"

	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
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

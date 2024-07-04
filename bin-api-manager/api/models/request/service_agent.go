package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// ParamServiceAgentCallsGET is rquest param define for
// GET /v1.0/service_agent/calls
type ParamServiceAgentCallsGET struct {
	Pagination
}

// BodyServiceAgentCallsPOST is rquest body define for
// POST /v1.0/service_agent/calls
type BodyServiceAgentCallsPOST struct {
	Source       commonaddress.Address   `json:"source" binding:"required"`
	Destinations []commonaddress.Address `json:"destinations" binding:"required"`
	FlowID       uuid.UUID               `json:"flow_id"`
	Actions      []fmaction.Action       `json:"actions"`
}

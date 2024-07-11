package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// ParamServiceAgentAgentsGET is rquest param define for
// GET /v1.0/service_agent/agents
type ParamServiceAgentAgentsGET struct {
	Pagination
}

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

// ParamServiceAgentChatroomsGET is rquest param define for
// GET /v1.0/service_agent/chatrooms
type ParamServiceAgentChatroomsGET struct {
	Pagination
}

// BodyServiceAgentChatroomsPOST is rquest body define for
// POST /v1.0/service_agents/chatrooms
type BodyServiceAgentChatroomsPOST struct {
	ParticipantID []uuid.UUID `json:"participant_ids"`
	Name          string      `json:"name"`
	Detail        string      `json:"detail"`
}

// BodyServiceAgentChatroomsIDPUT is rquest body define for
// PUT /v1.0/service_agents/chatrooms/<chatroom-id>
type BodyServiceAgentChatroomsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

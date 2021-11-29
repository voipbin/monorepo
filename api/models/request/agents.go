package request

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/address"
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/agent"
)

// ParamAgentsGET is rquest param define for GET /calls
type ParamAgentsGET struct {
	Pagination
	TagIDs string       `form:"tag_ids"`
	Status agent.Status `form:"status"`
}

// BodyAgentsPOST is rquest body define for POST /agents
type BodyAgentsPOST struct {
	Username   string `json:"username" binding:"required"`
	Password   string `json:"password" binding:"required"`
	Name       string `json:"name"`
	Detail     string `json:"detail"`
	RingMethod string `json:"ring_method"`

	Permission agent.Permission  `json:"permission"`
	TagIDs     []uuid.UUID       `json:"tag_ids"`
	Addresses  []address.Address `json:"addresses"`
}

// BodyAgentsIDPUT is rquest body define for PUT /agents/<agent-id>
type BodyAgentsIDPUT struct {
	Name       string `json:"name"`
	Detail     string `json:"detail"`
	RingMethod string `json:"ring_method"`
}

// BodyAgentsIDAddressesPUT is rquest body define for PUT /agents/<agent-id>/addresses
type BodyAgentsIDAddressesPUT struct {
	Addresses []address.Address `json:"addresses" binding:"required"`
}

// BodyAgentsIDTagIDsPUT is rquest body define for PUT /agents/<agent-id>/tag_ids
type BodyAgentsIDTagIDsPUT struct {
	TagIDs []uuid.UUID `json:"tag_ids" binding:"required"`
}

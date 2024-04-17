package request

import (
	"github.com/gofrs/uuid"
	amagent "gitlab.com/voipbin/bin-manager/agent-manager.git/models/agent"
	commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"
)

// ParamAgentsGET is rquest param define for
// GET /v1.0/agents
type ParamAgentsGET struct {
	Pagination
	TagIDs string         `form:"tag_ids"`
	Status amagent.Status `form:"status"`
}

// BodyAgentsPOST is rquest body define for
// POST /v1.0/agents
type BodyAgentsPOST struct {
	Username   string             `json:"username" binding:"required"`
	Password   string             `json:"password" binding:"required"`
	Name       string             `json:"name"`
	Detail     string             `json:"detail"`
	RingMethod amagent.RingMethod `json:"ring_method"`

	Permission amagent.Permission      `json:"permission"`
	TagIDs     []uuid.UUID             `json:"tag_ids"`
	Addresses  []commonaddress.Address `json:"addresses"`
}

// BodyAgentsIDPUT is rquest body define for
// PUT /v1.0/agents/<agent-id>
type BodyAgentsIDPUT struct {
	Name       string             `json:"name"`
	Detail     string             `json:"detail"`
	RingMethod amagent.RingMethod `json:"ring_method"`
}

// BodyAgentsIDAddressesPUT is rquest body define for
// PUT /v1.0/agents/<agent-id>/addresses
type BodyAgentsIDAddressesPUT struct {
	Addresses []commonaddress.Address `json:"addresses" binding:"required"`
}

// BodyAgentsIDTagIDsPUT is rquest body define for
// PUT /v1.0/agents/<agent-id>/tag_ids
type BodyAgentsIDTagIDsPUT struct {
	TagIDs []uuid.UUID `json:"tag_ids" binding:"required"`
}

// BodyAgentsIDStatusPUT is rquest body define for
// PUT /v1.0/agents/<agent-id>/status
type BodyAgentsIDStatusPUT struct {
	Status amagent.Status `json:"status" binding:"required"`
}

// BodyAgentsIDPermissionPUT is rquest body define for
// PUT /v1.0/agents/<agent-id>/permission
type BodyAgentsIDPermissionPUT struct {
	Permission amagent.Permission `json:"permission" binding:"required"`
}

// BodyAgentsIDPasswordPUT is rquest body define for
// PUT /v1.0/agents/<agent-id>/password
type BodyAgentsIDPasswordPUT struct {
	Password string `json:"password" binding:"required"`
}

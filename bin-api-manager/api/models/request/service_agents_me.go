package request

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	commonaddress "monorepo/bin-common-handler/models/address"
)

// BodyServiceAgentsMePUT is rquest body define for
// PUT /v1.0/service_agents/me
type BodyServiceAgentsMePUT struct {
	Name       string             `json:"name"`
	Detail     string             `json:"detail"`
	RingMethod amagent.RingMethod `json:"ring_method"`
}

// BodyServiceAgentsMeAddressesPUT is rquest body define for
// PUT /v1.0/service_agents/me/addresses
type BodyServiceAgentsMeAddressesPUT struct {
	Addresses []commonaddress.Address `json:"addresses" binding:"required"`
}

// BodyServiceAgentsMeStatusPUT is rquest body define for
// PUT /v1.0/service_agents/me/status
type BodyServiceAgentsMeStatusPUT struct {
	Status amagent.Status `json:"status" binding:"required"`
}

// BodyServiceAgentsMePasswordPUT is rquest body define for
// PUT /v1.0/service_agents/me/password
type BodyServiceAgentsMePasswordPUT struct {
	Password string `json:"password" binding:"required"`
}

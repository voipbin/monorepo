package account

import (
	commonbilling "monorepo/bin-common-handler/models/billing"
)

// PlanType define
type PlanType string

// list of plan types
const (
	PlanTypeFree         PlanType = "free"
	PlanTypeBasic        PlanType = "basic"
	PlanTypeProfessional PlanType = "professional"
	PlanTypeUnlimited    PlanType = "unlimited"
)

// PlanLimits defines the resource limits for a plan
type PlanLimits struct {
	Extensions     int
	Agents         int
	Queues         int
	Flows          int
	Conferences    int
	Trunks         int
	VirtualNumbers int
}

// GetLimit returns the limit for the given resource type.
// Returns 0 if the resource type is unknown.
func (p *PlanLimits) GetLimit(resourceType commonbilling.ResourceType) int {
	switch resourceType {
	case commonbilling.ResourceTypeExtension:
		return p.Extensions
	case commonbilling.ResourceTypeAgent:
		return p.Agents
	case commonbilling.ResourceTypeQueue:
		return p.Queues
	case commonbilling.ResourceTypeFlow:
		return p.Flows
	case commonbilling.ResourceTypeConference:
		return p.Conferences
	case commonbilling.ResourceTypeTrunk:
		return p.Trunks
	case commonbilling.ResourceTypeVirtualNumber:
		return p.VirtualNumbers
	default:
		return 0
	}
}

// PlanLimitMap maps plan types to their resource limits.
// A limit of 0 means unlimited (no restriction enforced).
var PlanLimitMap = map[PlanType]PlanLimits{
	PlanTypeFree:         {Extensions: 5, Agents: 5, Queues: 2, Flows: 5, Conferences: 2, Trunks: 1, VirtualNumbers: 10},
	PlanTypeBasic:        {Extensions: 50, Agents: 50, Queues: 10, Flows: 50, Conferences: 10, Trunks: 5, VirtualNumbers: 100},
	PlanTypeProfessional: {Extensions: 500, Agents: 500, Queues: 100, Flows: 500, Conferences: 100, Trunks: 50, VirtualNumbers: 1000},
	PlanTypeUnlimited:    {Extensions: 0, Agents: 0, Queues: 0, Flows: 0, Conferences: 0, Trunks: 0, VirtualNumbers: 0},
}

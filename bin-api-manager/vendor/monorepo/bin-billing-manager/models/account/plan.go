package account

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
func (p *PlanLimits) GetLimit(resourceType ResourceType) int {
	switch resourceType {
	case ResourceTypeExtension:
		return p.Extensions
	case ResourceTypeAgent:
		return p.Agents
	case ResourceTypeQueue:
		return p.Queues
	case ResourceTypeFlow:
		return p.Flows
	case ResourceTypeConference:
		return p.Conferences
	case ResourceTypeTrunk:
		return p.Trunks
	case ResourceTypeVirtualNumber:
		return p.VirtualNumbers
	default:
		return 0
	}
}

// PlanTokenMap maps plan types to their monthly token allowances.
// A value of 0 for Unlimited means bypass — unlimited tokens (no enforcement).
var PlanTokenMap = map[PlanType]int64{
	PlanTypeFree:         100,
	PlanTypeBasic:        1000,
	PlanTypeProfessional: 10000,
	PlanTypeUnlimited:    0,
}

// PlanLimitMap maps plan types to their resource limits.
// A limit of 0 means unlimited (no restriction enforced).
//
// Flows is set to 0 (unlimited) for all tiers because flow limits are now
// enforced locally in flow-manager with a hard cap, not through billing tiers.
// See bin-flow-manager/pkg/flowhandler/db.go maxFlowCount for details.
var PlanLimitMap = map[PlanType]PlanLimits{
	PlanTypeFree:         {Extensions: 5, Agents: 5, Queues: 2, Flows: 0, Conferences: 2, Trunks: 1, VirtualNumbers: 5},
	PlanTypeBasic:        {Extensions: 50, Agents: 50, Queues: 10, Flows: 0, Conferences: 10, Trunks: 5, VirtualNumbers: 50},
	PlanTypeProfessional: {Extensions: 500, Agents: 500, Queues: 100, Flows: 0, Conferences: 100, Trunks: 50, VirtualNumbers: 500},
	PlanTypeUnlimited:    {Extensions: 0, Agents: 0, Queues: 0, Flows: 0, Conferences: 0, Trunks: 0, VirtualNumbers: 0},
}

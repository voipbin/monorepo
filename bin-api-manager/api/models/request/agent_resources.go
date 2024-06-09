package request

// ParamAgentResourcesGET is rquest param define for
// GET /v1.0/agent_resources
type ParamAgentResourcesGET struct {
	Pagination
	ReferenceType string `form:"reference_type"`
}

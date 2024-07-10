package request

// ParamServiceAgentsConversationsGET is rquest param define for
// GET /v1.0/service_agent/conversations
type ParamServiceAgentsConversationsGET struct {
	Pagination
}

// ParamServiceAgentsConversationsIDMessagesGET is request param define for
// GET /v1.0/service_agents/conversations/<conversation-id>/messages
type ParamServiceAgentsConversationsIDMessagesGET struct {
	Pagination
}

// BodyServiceAgentsConversationsIDMessagesPOST is request body define for
// POST /v1.0/service_agents/conversations/<conversation-id>/messages
type BodyServiceAgentsConversationsIDMessagesPOST struct {
	Text string
}

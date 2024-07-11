package response

import (
	amagent "monorepo/bin-agent-manager/models/agent"
	cmcall "monorepo/bin-call-manager/models/call"
	cmgroupcall "monorepo/bin-call-manager/models/groupcall"
	chatchatroom "monorepo/bin-chat-manager/models/chatroom"
	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmessage "monorepo/bin-conversation-manager/models/message"
)

// BodyServiceAgentsAgentsGET is response body define for
// GET /v1.0/service_agent/agents
type BodyServiceAgentsAgentsGET struct {
	Result []*amagent.WebhookMessage `json:"result"`
	Pagination
}

// BodyServiceAgentsCallsGET is response body define for
// GET /v1.0/service_agent/calls
type BodyServiceAgentsCallsGET struct {
	Result []*cmcall.WebhookMessage `json:"result"`
	Pagination
}

// BodyServiceAgentsCallsPOST is response body define for
// POST /v1.0/service_agent/calls
type BodyServiceAgentsCallsPOST struct {
	Calls      []*cmcall.WebhookMessage      `json:"calls"`
	Groupcalls []*cmgroupcall.WebhookMessage `json:"groupcalls"`
}

// BodyServiceAgentsChatroomsGET is response body define for
// GET /v1.0/service_agent/chatrooms
type BodyServiceAgentsChatroomsGET struct {
	Result []*chatchatroom.WebhookMessage `json:"result"`
	Pagination
}

// BodyServiceAgentsConversationsGET is response body define for
// GET /v1.0/service_agent/conversations
type BodyServiceAgentsConversationsGET struct {
	Result []*cvconversation.WebhookMessage `json:"result"`
	Pagination
}

// BodyServiceAgentsChatroommessagesGET is response body define for
// GET /v1.0/service_agents/chatroommessages
type BodyServiceAgentsChatroommessagesGET struct {
	Result []*chatmessagechatroom.WebhookMessage `json:"result"`
	Pagination
}

// BodyServiceAgentsConversationsIDMessagesGET is rquest body define for
// GET /v1.0/service_agents/conversations/<conversation-id>/messages
type BodyServiceAgentsConversationsIDMessagesGET struct {
	Result []*cvmessage.WebhookMessage `json:"result"`
	Pagination
}

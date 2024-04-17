package response

import (
	cvconversation "monorepo/bin-conversation-manager/models/conversation"
	cvmessage "monorepo/bin-conversation-manager/models/message"
)

// BodyConversationsGET is rquest body define for
// GET /v1.0/conversations
type BodyConversationsGET struct {
	Result []*cvconversation.WebhookMessage `json:"result"`
	Pagination
}

// BodyConversationsIDMessagesGET is rquest body define for
// GET /v1.0/conversations/<conversation-id>/messages
type BodyConversationsIDMessagesGET struct {
	Result []*cvmessage.WebhookMessage `json:"result"`
	Pagination
}

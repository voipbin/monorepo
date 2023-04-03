package response

import (
	cvconversation "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/conversation"
	cvmessage "gitlab.com/voipbin/bin-manager/conversation-manager.git/models/message"
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

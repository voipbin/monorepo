package response

import (
	chatchatmessage "monorepo/bin-chat-manager/models/messagechat"
)

// BodyChatmessagesGET is rquest body define for
// GET /v1.0/chatmessages
type BodyChatmessagesGET struct {
	Result []*chatchatmessage.WebhookMessage `json:"result"`
	Pagination
}

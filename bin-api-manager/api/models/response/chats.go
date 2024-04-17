package response

import (
	chatchat "monorepo/bin-chat-manager/models/chat"
)

// BodyChatsGET is rquest body define for
// GET /v1.0/chats
type BodyChatsGET struct {
	Result []*chatchat.WebhookMessage `json:"result"`
	Pagination
}

package response

import (
	chatmessagechatroom "monorepo/bin-chat-manager/models/messagechatroom"
)

// BodyChatroommessagesGET is rquest body define for
// GET /v1.0/chatroommessages
type BodyChatroommessagesGET struct {
	Result []*chatmessagechatroom.WebhookMessage `json:"result"`
	Pagination
}

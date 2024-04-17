package response

import (
	chatchatroom "monorepo/bin-chat-manager/models/chatroom"
)

// BodyChatroomsGET is rquest body define for
// GET /v1.0/chatrooms
type BodyChatroomsGET struct {
	Result []*chatchatroom.WebhookMessage `json:"result"`
	Pagination
}

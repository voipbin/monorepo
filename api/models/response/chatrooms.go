package response

import (
	chatchatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chatroom"
)

// BodyChatroomsGET is rquest body define for
// GET /v1.0/chatrooms
type BodyChatroomsGET struct {
	Result []*chatchatroom.WebhookMessage `json:"result"`
	Pagination
}

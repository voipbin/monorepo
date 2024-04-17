package response

import (
	chatchat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
)

// BodyChatsGET is rquest body define for
// GET /v1.0/chats
type BodyChatsGET struct {
	Result []*chatchat.WebhookMessage `json:"result"`
	Pagination
}

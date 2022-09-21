package response

import (
	chatchat "gitlab.com/voipbin/bin-manager/chat-manager.git/models/chat"
)

// BodyChatsGET is rquest body define for GET /chats
type BodyChatsGET struct {
	Result []*chatchat.WebhookMessage `json:"result"`
	Pagination
}

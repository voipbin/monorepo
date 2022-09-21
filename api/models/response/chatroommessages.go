package response

import (
	chatmessagechatroom "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechatroom"
)

// BodyChatroommessagesGET is rquest body define for GET /chatroommessages
type BodyChatroommessagesGET struct {
	Result []*chatmessagechatroom.WebhookMessage `json:"result"`
	Pagination
}

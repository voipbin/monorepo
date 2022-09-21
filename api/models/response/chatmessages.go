package response

import (
	chatchatmessage "gitlab.com/voipbin/bin-manager/chat-manager.git/models/messagechat"
)

// BodyChatmessagesGET is rquest body define for GET /chatmessages
type BodyChatmessagesGET struct {
	Result []*chatchatmessage.WebhookMessage `json:"result"`
	Pagination
}

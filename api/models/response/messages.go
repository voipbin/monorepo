package response

import (
	mmmessage "gitlab.com/voipbin/bin-manager/message-manager.git/models/message"
)

// BodyMessagesGET is rquest body define for GET /messages
type BodyMessagesGET struct {
	Result []*mmmessage.WebhookMessage `json:"result"`
	Pagination
}

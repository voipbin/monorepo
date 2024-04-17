package response

import (
	mmmessage "monorepo/bin-message-manager/models/message"
)

// BodyMessagesGET is rquest body define for
// GET /v1.0/messages
type BodyMessagesGET struct {
	Result []*mmmessage.WebhookMessage `json:"result"`
	Pagination
}

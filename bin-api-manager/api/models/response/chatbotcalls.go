package response

import (
	chatbotchatbotcall "monorepo/bin-chatbot-manager/models/chatbotcall"
)

// BodyChatbotcallsGET is rquest body define for
// GET /v1.0/chatbotcalls
type BodyChatbotcallsGET struct {
	Result []*chatbotchatbotcall.WebhookMessage `json:"result"`
	Pagination
}

package response

import (
	chatbotchatbot "monorepo/bin-chatbot-manager/models/chatbot"
)

// BodyChatbotsGET is rquest body define for
// GET /v1.0/chatbots
type BodyChatbotsGET struct {
	Result []*chatbotchatbot.WebhookMessage `json:"result"`
	Pagination
}

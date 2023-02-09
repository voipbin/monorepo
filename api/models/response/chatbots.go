package response

import (
	chatbotchatbot "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
)

// BodyChatbotsGET is rquest body define for GET /chatbots
type BodyChatbotsGET struct {
	Result []*chatbotchatbot.WebhookMessage `json:"result"`
	Pagination
}

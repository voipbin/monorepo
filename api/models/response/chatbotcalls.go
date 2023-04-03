package response

import (
	chatbotchatbotcall "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbotcall"
)

// BodyChatbotcallsGET is rquest body define for
// GET /v1.0/chatbotcalls
type BodyChatbotcallsGET struct {
	Result []*chatbotchatbotcall.WebhookMessage `json:"result"`
	Pagination
}

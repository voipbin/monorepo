package request

import (
	chatbotchatbot "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
)

// BodyChatbotsPOST is rquest body define for POST /chatbots
type BodyChatbotsPOST struct {
	Name       string                    `json:"name"`
	Detail     string                    `json:"detail"`
	EngineType chatbotchatbot.EngineType `json:"engine_type"`
}

// ParamChatbotsGET is rquest param define for GET /chatbots
type ParamChatbotsGET struct {
	Pagination
}

package request

import (
	chatbotchatbot "gitlab.com/voipbin/bin-manager/chatbot-manager.git/models/chatbot"
)

// BodyChatbotsPOST is rquest body define for
// POST /v1.0/chatbots
type BodyChatbotsPOST struct {
	Name       string                    `json:"name"`
	Detail     string                    `json:"detail"`
	EngineType chatbotchatbot.EngineType `json:"engine_type"`
	InitPrompt string                    `json:"init_prompt"`
}

// ParamChatbotsGET is rquest param define for
// GET /v1.0/chatbots
type ParamChatbotsGET struct {
	Pagination
}

// BodyChatbotsIDPUT is rquest body define for
// PUT /v1.0/chatbots/<chatbot-id>
type BodyChatbotsIDPUT struct {
	Name       string                    `json:"name"`
	Detail     string                    `json:"detail"`
	EngineType chatbotchatbot.EngineType `json:"engine_type"`
	InitPrompt string                    `json:"init_prompt"`
}

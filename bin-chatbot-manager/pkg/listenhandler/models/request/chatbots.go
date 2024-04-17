package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-chatbot-manager/models/chatbot"
)

// V1DataChatbotsPost is
// v1 data type request struct for
// /v1/chatbots POST
type V1DataChatbotsPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Name       string    `json:"name"`
	Detail     string    `json:"detail"`

	EngineType chatbot.EngineType `json:"engine_type"`
	InitPrompt string             `json:"init_prompt"`
}

// V1DataChatbotsIDPut is
// v1 data type request struct for
// /v1/chatbots/<chatbot-id> PUT
type V1DataChatbotsIDPut struct {
	Name       string             `json:"name"`
	Detail     string             `json:"detail"`
	EngineType chatbot.EngineType `json:"engine_type"`
	InitPrompt string             `json:"init_prompt"`
}

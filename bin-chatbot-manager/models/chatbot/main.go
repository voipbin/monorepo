package chatbot

import (
	"github.com/gofrs/uuid"
)

// Chatbot define
type Chatbot struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	EngineType EngineType `json:"engine_type"`
	InitPrompt string     `json:"init_prompt"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// EngineType define
type EngineType string

// list of engine types
const (
	EngineTypeChatGPT EngineType = "chatGPT" // openai chatGPT. https://chat.openai.com/chat
	EngineTypeClova   EngineType = "clova"   // naver clova. https://www.ncloud.com/product/aiService/chatbot
)

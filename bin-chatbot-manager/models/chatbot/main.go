package chatbot

import (
	"monorepo/bin-common-handler/models/identity"
)

// Chatbot define
type Chatbot struct {
	identity.Identity

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	EngineType  EngineType  `json:"engine_type,omitempty"`
	EngineModel EngineModel `json:"engine_model,omitempty"`
	InitPrompt  string      `json:"init_prompt,omitempty"`

	CredentialBase64    string `json:"credential_base64,omitempty"`
	CredentialProjectID string `json:"credential_project_id,omitempty"`

	// timestamp
	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// EngineType define
type EngineType string

// list of engine types
const (
	EngineTypeChatGPT    EngineType = "chatGPT"    // openai chatGPT. https://chat.openai.com/chat
	EngineTypeDialogFlow EngineType = "dialogflow" // google dialogflow. https://cloud.google.com/dialogflow
	EngineTypeClova      EngineType = "clova"      // naver clova. https://www.ncloud.com/product/aiService/chatbot
)

type EngineModel string

// list of engine models
const (
	EngineModelChatGPTO1Mini         EngineModel = "o1-mini"
	EngineModelChatGPTO1Preview      EngineModel = "o1-preview"
	EngineModelChatGPTO1             EngineModel = "o1"
	EngineModelChatGPTO3Mini         EngineModel = "o3-mini"
	EngineModelChatGPT4O             EngineModel = "gpt-4o"
	EngineModelChatGPT4OMini         EngineModel = "gpt-4o-mini"
	EngineModelChatGPT4Turbo         EngineModel = "gpt-4-turbo"
	EngineModelChatGPT4VisionPreview EngineModel = "gpt-4-vision-preview"
	EngineModelChatGPT4              EngineModel = "gpt-4"
	EngineModelChatGPT3Dot5Turbo     EngineModel = "gpt-3.5-turbo"
)

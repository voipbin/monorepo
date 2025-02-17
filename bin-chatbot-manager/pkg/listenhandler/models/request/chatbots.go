package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-chatbot-manager/models/chatbot"
)

// V1DataChatbotsPost is
// v1 data type request struct for
// /v1/chatbots POST
type V1DataChatbotsPost struct {
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	Name       string    `json:"name,omitempty"`
	Detail     string    `json:"detail,omitempty"`

	EngineType  chatbot.EngineType  `json:"engine_type,omitempty"`
	EngineModel chatbot.EngineModel `json:"engine_model,omitempty"`
	InitPrompt  string              `json:"init_prompt,omitempty"`

	CredentialBase64    string `json:"credential_base64,omitempty"`
	CredentialProjectID string `json:"credential_project_id,omitempty"`
}

// V1DataChatbotsIDPut is
// v1 data type request struct for
// /v1/chatbots/<chatbot-id> PUT
type V1DataChatbotsIDPut struct {
	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	EngineType  chatbot.EngineType  `json:"engine_type,omitempty"`
	EngineModel chatbot.EngineModel `json:"engine_model,omitempty"`
	InitPrompt  string              `json:"init_prompt,omitempty"`

	CredentialBase64    string `json:"credential_base64,omitempty"`
	CredentialProjectID string `json:"credential_project_id,omitempty"`
}

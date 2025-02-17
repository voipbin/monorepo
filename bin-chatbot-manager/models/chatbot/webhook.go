package chatbot

import (
	"encoding/json"
	"monorepo/bin-common-handler/models/identity"
)

// WebhookMessage defines webhook event
type WebhookMessage struct {
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

// ConvertWebhookMessage converts to the event
func (h *Chatbot) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Name:   h.Name,
		Detail: h.Detail,

		EngineType:  h.EngineType,
		EngineModel: h.EngineModel,
		InitPrompt:  h.InitPrompt,

		CredentialBase64:    h.CredentialBase64,
		CredentialProjectID: h.CredentialProjectID,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Chatbot) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

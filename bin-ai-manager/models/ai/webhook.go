package ai

import (
	"encoding/json"
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage defines webhook event
type WebhookMessage struct {
	commonidentity.Identity

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	EngineType  EngineType     `json:"engine_type,omitempty"`
	EngineModel EngineModel    `json:"engine_model,omitempty"`
	EngineData  map[string]any `json:"engine_data,omitempty"`
	EngineKey   string         `json:"engine_key,omitempty"`

	InitPrompt string `json:"init_prompt,omitempty"`

	// timestamp
	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts to the event
func (h *AI) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Name:   h.Name,
		Detail: h.Detail,

		EngineType:  h.EngineType,
		EngineModel: h.EngineModel,
		EngineData:  h.EngineData,
		EngineKey:   h.EngineKey,

		InitPrompt: h.InitPrompt,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *AI) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

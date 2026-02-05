package ai

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-ai-manager/models/tool"
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

	TTSType    TTSType `json:"tts_type,omitempty"`
	TTSVoiceID string  `json:"tts_voice_id,omitempty"`

	STTType STTType `json:"stt_type,omitempty"`

	ToolNames []tool.ToolName `json:"tool_names,omitempty"`

	// timestamp
	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
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

		TTSType:    h.TTSType,
		TTSVoiceID: h.TTSVoiceID,

		STTType: h.STTType,

		ToolNames: h.ToolNames,

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

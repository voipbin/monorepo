package ai

import (
	"encoding/json"
	"time"

	"github.com/gofrs/uuid"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"monorepo/bin-ai-manager/models/tool"
)

// WebhookMessage defines webhook event
type WebhookMessage struct {
	commonidentity.Identity

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	EngineModel EngineModel    `json:"engine_model,omitempty"`
	Parameter   map[string]any `json:"parameter,omitempty"`
	EngineKey   string         `json:"engine_key,omitempty"`
	RagID       uuid.UUID      `json:"rag_id,omitempty"`

	InitPrompt             string    `json:"init_prompt,omitempty"`
	CurrentPromptHistoryID uuid.UUID `json:"current_prompt_history_id"`

	TTSType    TTSType `json:"tts_type,omitempty"`
	TTSVoiceID string  `json:"tts_voice_id,omitempty"`

	STTType          STTType    `json:"stt_type,omitempty"`
	STTLanguage      string     `json:"stt_language,omitempty"`
	VADConfig        *VADConfig `json:"vad_config,omitempty"`
	SmartTurnEnabled bool       `json:"smart_turn_enabled,omitempty"`

	ToolNames []tool.ToolName `json:"tool_names,omitempty"`

	DirectHash string `json:"direct_hash,omitempty"`

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

		EngineModel: h.EngineModel,
		Parameter:   h.Parameter,
		EngineKey:   h.EngineKey,
		RagID:       h.RagID,

		InitPrompt:             h.InitPrompt,
		CurrentPromptHistoryID: h.CurrentPromptHistoryID,

		TTSType:    h.TTSType,
		TTSVoiceID: h.TTSVoiceID,

		STTType:          h.STTType,
		STTLanguage:      h.STTLanguage,
		VADConfig:        h.VADConfig,
		SmartTurnEnabled: h.SmartTurnEnabled,

		ToolNames: h.ToolNames,

		DirectHash: h.DirectHash,

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

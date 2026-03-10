package aicall

import (
	"encoding/json"
	"time"

	"monorepo/bin-ai-manager/models/ai"
	"monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines webhook event
type WebhookMessage struct {
	identity.Identity

	AssistanceType AssistanceType `json:"assistance_type,omitempty"`
	AssistanceID   uuid.UUID      `json:"assistance_id,omitempty"`

	AIEngineModel ai.EngineModel `json:"ai_engine_model,omitempty"`
	AITTSType     ai.TTSType     `json:"ai_tts_type,omitempty"`
	AITTSVoiceID  string         `json:"ai_tts_voice_id,omitempty"`
	AISTTType     ai.STTType     `json:"ai_stt_type,omitempty"`
	AIVADConfig        *ai.VADConfig  `json:"ai_vad_config,omitempty"`
	AISmartTurnEnabled bool           `json:"ai_smart_turn_enabled,omitempty"`

	Parameter map[string]any `json:"parameter,omitempty"`

	ActiveflowID  uuid.UUID     `json:"activeflow_id,omitempty"`
	ReferenceType ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID     `json:"reference_id,omitempty"`

	ConfbridgeID uuid.UUID `json:"confbridge_id,omitempty"`

	Status Status `json:"status,omitempty"`

	Gender   Gender `json:"gender,omitempty"`
	Language string `json:"language,omitempty"`

	TMEnd    *time.Time `json:"tm_end"`
	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *AIcall) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		AssistanceType: h.AssistanceType,
		AssistanceID:   h.AssistanceID,

		AIEngineModel: h.AIEngineModel,
		AITTSType:     h.AITTSType,
		AITTSVoiceID:  h.AITTSVoiceID,
		AISTTType:     h.AISTTType,
		AIVADConfig:        h.AIVADConfig,
		AISmartTurnEnabled: h.AISmartTurnEnabled,

		Parameter: h.Parameter,

		ActiveflowID:  h.ActiveflowID,
		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		ConfbridgeID: h.ConfbridgeID,

		Status: h.Status,

		Gender:   h.Gender,
		Language: h.Language,

		TMEnd:    h.TMEnd,
		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *AIcall) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

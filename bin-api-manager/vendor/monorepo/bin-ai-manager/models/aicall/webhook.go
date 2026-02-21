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

	AIID          uuid.UUID      `json:"ai_id,omitempty"`
	AIEngineType  ai.EngineType  `json:"ai_engine_type,omitempty"`
	AIEngineModel ai.EngineModel `json:"ai_engine_model,omitempty"`

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

		AIID:          h.AIID,
		AIEngineType:  h.AIEngineType,
		AIEngineModel: h.AIEngineModel,

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

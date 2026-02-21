package speaking

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
	"monorepo/bin-tts-manager/models/streaming"

	"github.com/gofrs/uuid"
)

// WebhookMessage is the external-facing representation of a Speaking session.
// It omits internal fields like PodID that should not be exposed to API clients.
type WebhookMessage struct {
	commonidentity.Identity

	ReferenceType streaming.ReferenceType `json:"reference_type,omitempty"`
	ReferenceID   uuid.UUID               `json:"reference_id,omitempty"`
	Language      string                  `json:"language,omitempty"`
	Provider      string                  `json:"provider,omitempty"`
	VoiceID       string                  `json:"voice_id,omitempty"`
	Direction     streaming.Direction     `json:"direction,omitempty"`
	Status        Status                  `json:"status,omitempty"`

	TMCreate *time.Time `json:"tm_create,omitempty"`
	TMUpdate *time.Time `json:"tm_update,omitempty"`
	TMDelete *time.Time `json:"tm_delete,omitempty"`
}

// ConvertWebhookMessage converts a Speaking to its external-facing representation.
func (h *Speaking) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,
		Language:      h.Language,
		Provider:      h.Provider,
		VoiceID:       h.VoiceID,
		Direction:     h.Direction,
		Status:        h.Status,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates webhook event data.
func (h *Speaking) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

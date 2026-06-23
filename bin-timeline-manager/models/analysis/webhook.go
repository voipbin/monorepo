package analysis

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage is the customer-facing projection of an Analysis (design §7.1,
// review F5). It deliberately DROPS internal fields so the public API is not
// coupled to vendor/infra naming:
//   - Model (raw engine/stage model id): dropped.
//   - the gateway finish_reason / token counts: never persisted here, not exposed.
//
// Error is the sanitized operator-safe string only (the orchestrator never
// persists raw provider errors or stack traces). Result is the structured
// verdict JSON (already has evidence_index resolved; no internal-only fields).
type WebhookMessage struct {
	commonidentity.Identity

	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`

	Status Status          `json:"status"`
	Result json.RawMessage `json:"result,omitempty"`
	Error  string          `json:"error,omitempty"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts the internal Analysis to its customer-facing
// projection, dropping Model (review F5).
func (h *Analysis) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		ActiveflowID: h.ActiveflowID,

		Status: h.Status,
		Result: h.Result,
		Error:  h.Error,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates the marshaled webhook payload.
func (h *Analysis) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

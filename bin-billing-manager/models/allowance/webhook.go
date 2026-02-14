package allowance

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines the API-safe representation of an Allowance.
type WebhookMessage struct {
	commonidentity.Identity

	AccountID  uuid.UUID  `json:"account_id"`
	CycleStart *time.Time `json:"cycle_start"`
	CycleEnd   *time.Time `json:"cycle_end"`

	TokensTotal int `json:"tokens_total"`
	TokensUsed  int `json:"tokens_used"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the API-safe webhook message.
func (h *Allowance) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		AccountID:  h.AccountID,
		CycleStart: h.CycleStart,
		CycleEnd:   h.CycleEnd,

		TokensTotal: h.TokensTotal,
		TokensUsed:  h.TokensUsed,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generates a WebhookEvent byte payload.
func (h *Allowance) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

package billing

import (
	"encoding/json"
	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	AccountID uuid.UUID `json:"account_id"` // billing account

	Status Status `json:"status"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	CostPerUnit float32 `json:"cost_per_unit"`
	CostTotal   float32 `json:"cost_total"`

	BillingUnitCount float32 `json:"billing_unit_count"`

	TMBillingStart string `json:"tm_billing_start"`
	TMBillingEnd   string `json:"tm_billing_end"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Billing) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		AccountID: h.AccountID,

		Status: h.Status,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		CostPerUnit: h.CostPerUnit,
		CostTotal:   h.CostTotal,

		BillingUnitCount: h.BillingUnitCount,

		TMBillingStart: h.TMBillingStart,
		TMBillingEnd:   h.TMBillingEnd,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Billing) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

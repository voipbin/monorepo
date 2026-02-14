package billing

import (
	"encoding/json"
	"time"

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

	CostType          CostType `json:"cost_type"`
	CostUnitCount     float32  `json:"cost_unit_count"`
	CostTokenPerUnit  int      `json:"cost_token_per_unit"`
	CostTokenTotal    int      `json:"cost_token_total"`
	CostCreditPerUnit float32  `json:"cost_credit_per_unit"`
	CostCreditTotal   float32  `json:"cost_credit_total"`

	TMBillingStart *time.Time `json:"tm_billing_start"`
	TMBillingEnd   *time.Time `json:"tm_billing_end"`

	// timestamp
	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Billing) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		AccountID: h.AccountID,

		Status: h.Status,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		CostType:          h.CostType,
		CostUnitCount:     h.CostUnitCount,
		CostTokenPerUnit:  h.CostTokenPerUnit,
		CostTokenTotal:    h.CostTokenTotal,
		CostCreditPerUnit: h.CostCreditPerUnit,
		CostCreditTotal:   h.CostCreditTotal,

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

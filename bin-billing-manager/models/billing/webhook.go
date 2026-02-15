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

	AccountID uuid.UUID `json:"account_id"`

	TransactionType TransactionType `json:"transaction_type"`
	Status          Status          `json:"status"`

	ReferenceType ReferenceType `json:"reference_type"`
	ReferenceID   uuid.UUID     `json:"reference_id"`

	CostType      CostType `json:"cost_type"`
	UsageDuration int      `json:"usage_duration"`
	BillableUnits int      `json:"billable_units"`

	RateTokenPerUnit  int64 `json:"rate_token_per_unit"`
	RateCreditPerUnit int64 `json:"rate_credit_per_unit"`

	AmountToken  int64 `json:"amount_token"`
	AmountCredit int64 `json:"amount_credit"`

	BalanceTokenSnapshot  int64 `json:"balance_token_snapshot"`
	BalanceCreditSnapshot int64 `json:"balance_credit_snapshot"`

	IdempotencyKey uuid.UUID `json:"idempotency_key"`

	TMBillingStart *time.Time `json:"tm_billing_start"`
	TMBillingEnd   *time.Time `json:"tm_billing_end"`

	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Billing) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		AccountID: h.AccountID,

		TransactionType: h.TransactionType,
		Status:          h.Status,

		ReferenceType: h.ReferenceType,
		ReferenceID:   h.ReferenceID,

		CostType:      h.CostType,
		UsageDuration: h.UsageDuration,
		BillableUnits: h.BillableUnits,

		RateTokenPerUnit:  h.RateTokenPerUnit,
		RateCreditPerUnit: h.RateCreditPerUnit,

		AmountToken:  h.AmountToken,
		AmountCredit: h.AmountCredit,

		BalanceTokenSnapshot:  h.BalanceTokenSnapshot,
		BalanceCreditSnapshot: h.BalanceCreditSnapshot,

		IdempotencyKey: h.IdempotencyKey,

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

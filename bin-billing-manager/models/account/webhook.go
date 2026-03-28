package account

import (
	"encoding/json"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// WebhookMessage defines
type WebhookMessage struct {
	commonidentity.Identity

	Name   string `json:"name"`
	Detail string `json:"detail"`

	PlanType   PlanType   `json:"plan_type"`
	PlanStatus PlanStatus `json:"plan_status"`

	BalanceCredit int64 `json:"balance_credit"`
	BalanceToken  int64 `json:"balance_token"`

	PaymentType   PaymentType   `json:"payment_type"`
	PaymentMethod PaymentMethod `json:"payment_method"`

	PaddleSubscriptionID string `json:"paddle_subscription_id,omitempty"`
	PaddleCustomerID     string `json:"paddle_customer_id,omitempty"`

	TmLastTopUp *time.Time `json:"tm_last_topup"`
	TmNextTopUp *time.Time `json:"tm_next_topup"`

	// timestamp
	TMCreate *time.Time `json:"tm_create"`
	TMUpdate *time.Time `json:"tm_update"`
	TMDelete *time.Time `json:"tm_delete"`
}

// ConvertWebhookMessage converts to the event
func (h *Account) ConvertWebhookMessage() *WebhookMessage {
	return &WebhookMessage{
		Identity: h.Identity,

		Name:   h.Name,
		Detail: h.Detail,

		PlanType:   h.PlanType,
		PlanStatus: h.PlanStatus,

		BalanceCredit: h.BalanceCredit,
		BalanceToken:  h.BalanceToken,

		PaymentType:   h.PaymentType,
		PaymentMethod: h.PaymentMethod,

		PaddleSubscriptionID: h.PaddleSubscriptionID,
		PaddleCustomerID:     h.PaddleCustomerID,

		TmLastTopUp: h.TmLastTopUp,
		TmNextTopUp: h.TmNextTopUp,

		TMCreate: h.TMCreate,
		TMUpdate: h.TMUpdate,
		TMDelete: h.TMDelete,
	}
}

// CreateWebhookEvent generate WebhookEvent
func (h *Account) CreateWebhookEvent() ([]byte, error) {
	e := h.ConvertWebhookMessage()

	m, err := json.Marshal(e)
	if err != nil {
		return nil, err
	}

	return m, nil
}

package account

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Account define
type Account struct {
	commonidentity.Identity

	Status Status `json:"status" db:"status"`

	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	PlanType   PlanType   `json:"plan_type" db:"plan_type"`
	PlanStatus PlanStatus `json:"plan_status" db:"plan_status"`

	BalanceCredit int64 `json:"balance_credit" db:"balance_credit"`
	BalanceToken  int64 `json:"balance_token" db:"balance_token"`

	PaymentType   PaymentType   `json:"payment_type" db:"payment_type"`
	PaymentMethod PaymentMethod `json:"payment_method" db:"payment_method"`

	PaddleSubscriptionID string `json:"paddle_subscription_id" db:"paddle_subscription_id"`
	PaddleCustomerID     string `json:"paddle_customer_id" db:"paddle_customer_id"`

	TmLastTopUp *time.Time `json:"tm_last_topup" db:"tm_last_topup"`
	TmNextTopUp *time.Time `json:"tm_next_topup" db:"tm_next_topup"`

	// timestamp
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// EventSubscriptionID returns the subscription address of this type on the global topic
// exchange `bin-manager.event`: the resource's own id (VOIP-1404 §4.2, VOIP-1419).
func (h *Account) EventSubscriptionID() string {
	return h.ID.String()
}

// Status defines the account status
type Status string

// list of Statuses
const (
	StatusActive  Status = "active"
	StatusFrozen  Status = "frozen"
	StatusDeleted Status = "deleted"
)

// PaymentType define
type PaymentType string

// list of PaymentTypes
const (
	PaymentTypeNone    PaymentType = ""
	PaymentTypePrepaid PaymentType = "prepaid"
)

// PaymentMethod define
type PaymentMethod string

// list of PaymentMethods
const (
	PaymentMethodNone       PaymentMethod = ""
	PaymentMethodCreditCard PaymentMethod = "credit card"
)

// PlanStatus defines the subscription plan status
type PlanStatus string

// list of PlanStatuses
const (
	PlanStatusActive    PlanStatus = "active"
	PlanStatusCanceling PlanStatus = "canceling"
)

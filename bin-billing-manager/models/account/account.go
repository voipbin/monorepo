package account

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Account define
type Account struct {
	commonidentity.Identity

	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	PlanType PlanType `json:"plan_type" db:"plan_type"`

	BalanceCredit int64 `json:"balance_credit" db:"balance_credit"`
	BalanceToken  int64 `json:"balance_token" db:"balance_token"`

	PaymentType   PaymentType   `json:"payment_type" db:"payment_type"`
	PaymentMethod PaymentMethod `json:"payment_method" db:"payment_method"`

	TmLastTopUp *time.Time `json:"tm_last_topup" db:"tm_last_topup"`
	TmNextTopUp *time.Time `json:"tm_next_topup" db:"tm_next_topup"`

	// timestamp
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

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

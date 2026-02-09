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

	Type     Type     `json:"type" db:"type"`
	PlanType PlanType `json:"plan_type" db:"plan_type"`

	Balance float32 `json:"balance" db:"balance"` // USD

	PaymentType   PaymentType   `json:"payment_type" db:"payment_type"`
	PaymentMethod PaymentMethod `json:"payment_method" db:"payment_method"`

	// timestamp
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// Type define
type Type string

// list of types
const (
	TypeAdmin  Type = "admin"  // admin type
	TypeNormal Type = "normal" // normal type
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

package account

import "github.com/gofrs/uuid"

// Account define
type Account struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Type Type `json:"type"`

	Balance float32 `json:"balance"` // USD

	PaymentType   PaymentType   `json:"payment_type"`
	PaymentMethod PaymentMethod `json:"payment_method"`

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
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

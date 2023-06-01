package account

import (
	"github.com/gofrs/uuid"
)

// Account defines
type Account struct {
	ID         uuid.UUID `json:"id"`          // customer id
	CustomerID uuid.UUID `json:"customer_id"` // customer

	Type Type `json:"type"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Secret string `json:"secret"` // secret
	Token  string `json:"token"`  // usually api token

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Type defines
type Type string

// list of types
const (
	TypeLine Type = "line"
	TypeSMS  Type = "sms"
)

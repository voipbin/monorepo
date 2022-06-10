package account

import (
	"github.com/gofrs/uuid"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
)

// Account defines
type Account struct {
	ID uuid.UUID `json:"id"` // customer id

	// line
	LineSecret string `json:"line_secret"`
	LineToken  string `json:"line_token"`
}

// CreateAccountFromCustomer creates messagetarget using the cscustomer.Customer
func CreateAccountFromCustomer(cs *cscustomer.Customer) *Account {
	return &Account{
		ID: cs.ID,

		LineSecret: "",
		LineToken:  "",
	}
}

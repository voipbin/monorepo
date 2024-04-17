package account

import (
	cscustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"

	"monorepo/bin-webhook-manager/models/webhook"
)

// Account defines
type Account struct {
	ID            uuid.UUID          `json:"id"`
	WebhookMethod webhook.MethodType `json:"webhook_method"`
	WebhookURI    string             `json:"webhook_uri"`
}

// CreateAccountFromCustomer creates account from the cscustomer.Customer
func CreateAccountFromCustomer(cs *cscustomer.Customer) *Account {
	return &Account{
		ID:            cs.ID,
		WebhookMethod: webhook.MethodType(cs.WebhookMethod),
		WebhookURI:    cs.WebhookURI,
	}
}

package account

import (
	"github.com/gofrs/uuid"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"

	"gitlab.com/voipbin/bin-manager/webhook-manager.git/models/webhook"
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

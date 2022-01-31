package messagetarget

import (
	"github.com/gofrs/uuid"
	cscustomer "gitlab.com/voipbin/bin-manager/customer-manager.git/models/customer"
)

// MessageTarget defines
type MessageTarget struct {
	ID            uuid.UUID `json:"id"`
	WebhookMethod string    `json:"webhook_method"`
	WebhookURI    string    `json:"webhook_uri"`
}

// CreateMessageTargetFromCustomer creates messagetarget using the cscustomer.Customer
func CreateMessageTargetFromCustomer(cs *cscustomer.Customer) *MessageTarget {
	return &MessageTarget{
		ID:            cs.ID,
		WebhookMethod: cs.WebhookMethod,
		WebhookURI:    cs.WebhookURI,
	}
}

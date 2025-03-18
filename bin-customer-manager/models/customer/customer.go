package customer

import "github.com/gofrs/uuid"

// Customer defines
type Customer struct {
	ID uuid.UUID `json:"id"` // Customer's ID

	Name   string `json:"name,omitempty"`   //  name
	Detail string `json:"detail,omitempty"` //  detail

	Email       string `json:"email,omitempty"`
	PhoneNumber string `json:"phone_number,omitempty"`
	Address     string `json:"address,omitempty"`

	// webhook info
	WebhookMethod WebhookMethod `json:"webhook_method,omitempty"` // webhook method
	WebhookURI    string        `json:"webhook_uri,omitempty"`    // webhook uri

	BillingAccountID uuid.UUID `json:"billing_account_id,omitempty"` // default billing account id

	TMCreate string `json:"tm_create,omitempty"` // Created timestamp.
	TMUpdate string `json:"tm_update,omitempty"` // Updated timestamp.
	TMDelete string `json:"tm_delete,omitempty"` // Deleted timestamp.
}

// WebhookMethod defines
type WebhookMethod string

// list of methods
const (
	WebhookMethodNone   = ""
	WebhookMethodPost   = "POST"
	WebhookMethodGet    = "GET"
	WebhookMethodPut    = "PUT"
	WebhookMethodDelete = "DELETE"
)

var (
	IDEmpty = uuid.FromStringOrNil("00000000-0000-0000-0000-00000000000") //

	// voipbin internal service's customer id
	IDCallManager = uuid.FromStringOrNil("00000000-0000-0000-0001-00000000001")
	IDAIManager   = uuid.FromStringOrNil("00000000-0000-0000-0001-00000000002")
)

package customer

import (
	"time"

	"github.com/gofrs/uuid"
)

// Status represents the account lifecycle state
type Status string

const (
	StatusActive  Status = "active"
	StatusFrozen  Status = "frozen"
	StatusDeleted Status = "deleted"
)

// Customer defines
type Customer struct {
	ID uuid.UUID `json:"id" db:"id,uuid"` // Customer's ID

	Name   string `json:"name,omitempty" db:"name"`     //  name
	Detail string `json:"detail,omitempty" db:"detail"` //  detail

	Email       string `json:"email,omitempty" db:"email"`
	PhoneNumber string `json:"phone_number,omitempty" db:"phone_number"`
	Address     string `json:"address,omitempty" db:"address"`

	// webhook info
	WebhookMethod WebhookMethod `json:"webhook_method,omitempty" db:"webhook_method"` // webhook method
	WebhookURI    string        `json:"webhook_uri,omitempty" db:"webhook_uri"`       // webhook uri

	BillingAccountID uuid.UUID `json:"billing_account_id,omitempty" db:"billing_account_id,uuid"` // default billing account id

	EmailVerified bool `json:"email_verified" db:"email_verified"`

	Status              Status     `json:"status" db:"status"`
	TMDeletionScheduled *time.Time `json:"tm_deletion_scheduled" db:"tm_deletion_scheduled"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"` // Created timestamp.
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"` // Updated timestamp.
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"` // Deleted timestamp.
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

	// GuestCustomerID is the guest/demo account customer id.
	GuestCustomerID = uuid.FromStringOrNil("a856c986-4b06-4496-9641-4d0ecbc67df5")
)

package customer

import "github.com/gofrs/uuid"

// Customer defines
type Customer struct {
	ID uuid.UUID `json:"id"` // Customer's ID

	Username     string `json:"username,omitempty"` // Customer's username
	PasswordHash string `json:"-"`                  // Hashed Password

	Name   string `json:"name,omitempty"`   //  name
	Detail string `json:"detail,omitempty"` //  detail

	// webhook info
	WebhookMethod WebhookMethod `json:"webhook_method,omitempty"` // webhook method
	WebhookURI    string        `json:"webhook_uri,omitempty"`    // webhook uri

	PermissionIDs []uuid.UUID `json:"permission_ids,omitempty"` // customer's permission ids

	BillingAccountID uuid.UUID `json:"billing_account_id"` // default billing account id

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

// HasPermission returns true if the customer has the given permission
func (h *Customer) HasPermission(perm uuid.UUID) bool {
	for _, item := range h.PermissionIDs {
		if item == perm {
			return true
		}
	}
	return false
}

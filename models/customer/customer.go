package customer

import "github.com/gofrs/uuid"

// Customer defines
type Customer struct {
	ID uuid.UUID `json:"id"` // Customer's ID

	Username     string `json:"username"` // Customer's username
	PasswordHash string `json:"-"`        // Hashed Password

	Name          string        `json:"name"`           //  name
	Detail        string        `json:"detail"`         //  detail
	WebhookMethod WebhookMethod `json:"webhook_method"` // webhook method
	WebhookURI    string        `json:"webhook_uri"`    // webhook uri

	PermissionIDs []uuid.UUID `json:"permission_ids"` // customer's permission ids

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
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

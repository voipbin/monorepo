package customer

import "github.com/gofrs/uuid"

// FieldStruct defines allowed filters for Customer queries
// Each field corresponds to a filterable database column
type FieldStruct struct {
	ID                uuid.UUID `filter:"id"`
	Name              string    `filter:"name"`
	Email             string    `filter:"email"`
	PhoneNumber       string    `filter:"phone_number"`
	BillingAccountID  uuid.UUID `filter:"billing_account_id"`
	Deleted           bool      `filter:"deleted"`
}

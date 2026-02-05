package contact

import (
	"time"

	"github.com/gofrs/uuid"
)

// Email represents an email address associated with a contact.
// Contacts can have multiple email addresses (work, personal, etc.).
//
// Email addresses are stored in lowercase for case-insensitive matching.
// The email format is validated on input but not verified for deliverability.
type Email struct {
	// ID is the unique identifier for this email record.
	ID uuid.UUID `json:"id" db:"id,uuid"`

	// CustomerID ensures tenant isolation. Must match the parent contact's
	// customer ID. Enables efficient queries filtered by customer.
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`

	// ContactID references the parent contact that owns this email.
	// When the contact is deleted, associated emails are cascade deleted.
	ContactID uuid.UUID `json:"contact_id" db:"contact_id,uuid"`

	// Address is the email address in lowercase format.
	// Stored lowercase for case-insensitive matching during lookup.
	// Example: "john.smith@example.com"
	Address string `json:"address" db:"address"`

	// Type categorizes the email address for organizational purposes.
	// Valid values:
	//   - "work": Work/business email
	//   - "personal": Personal/private email
	//   - "other": Any other type
	// Empty string is allowed and treated as unspecified.
	Type string `json:"type" db:"type"`

	// IsPrimary indicates this is the preferred/default email address.
	// Used when a single email is needed (e.g., sending notifications).
	// Only one email per contact should be marked as primary.
	IsPrimary bool `json:"is_primary" db:"is_primary"`

	// TMCreate is when this email was added to the contact.
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
}

// Email type constants
const (
	EmailTypeWork     string = "work"
	EmailTypePersonal string = "personal"
	EmailTypeOther    string = "other"
)

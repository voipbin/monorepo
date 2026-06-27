package contact

import (
	"time"

	"github.com/gofrs/uuid"
)

// PhoneNumber represents a phone number associated with a contact.
// Contacts can have multiple phone numbers (mobile, work, home, etc.).
//
// Phone numbers are persisted in the unified contact_addresses table as
// (type='tel', target=<E.164>) rows. The Number field carries the normalized
// E.164 string (no separate raw/normalized split). For example, both
// "(555) 123-4567" and "+1-555-123-4567" are stored as "+155****4567".
type PhoneNumber struct {
	// ID is the unique identifier for this phone number record.
	ID uuid.UUID `json:"id" db:"id,uuid"`

	// CustomerID ensures tenant isolation. Must match the parent contact's
	// customer ID. Enables efficient queries filtered by customer.
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`

	// ContactID references the parent contact that owns this phone number.
	// When the contact is deleted, associated phone numbers are cascade deleted.
	ContactID uuid.UUID `json:"contact_id" db:"contact_id,uuid"`

	// Number stores the phone number in normalized E.164 format. E.164 is the
	// international standard: + followed by country code and subscriber number,
	// with no spaces, dashes, or parentheses. The value maps to the
	// contact_addresses.target column for tel-type rows and is used for:
	//   - Lookup operations (e.g., finding contact by caller ID)
	//   - Deduplication (prevent adding the same number twice)
	//   - Integration with telephony systems
	//
	// Examples: "+155****4567" (US), "+442****4567" (UK), "+813****5678" (Japan)
	Number string `json:"number" db:"number"`

	// Type categorizes the phone number for organizational purposes.
	// Valid values:
	//   - "mobile": Cell/mobile phone (preferred for SMS)
	//   - "work": Work/office phone
	//   - "home": Home/residential phone
	//   - "fax": Fax number
	//   - "other": Any other type
	// Empty string is allowed and treated as unspecified.
	Type string `json:"type" db:"type"`

	// IsPrimary indicates this is the preferred/default phone number.
	// Used when a single phone number is needed (e.g., click-to-call).
	// Only one phone number per contact should be marked as primary.
	// If multiple are marked, the first one found is used.
	IsPrimary bool `json:"is_primary" db:"is_primary"`

	// TMCreate is when this phone number was added to the contact.
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
}

// PhoneNumber type constants
const (
	PhoneTypeMobile string = "mobile"
	PhoneTypeWork   string = "work"
	PhoneTypeHome   string = "home"
	PhoneTypeFax    string = "fax"
	PhoneTypeOther  string = "other"
)

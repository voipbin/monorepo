package contact

import (
	"time"

	"github.com/gofrs/uuid"
)

// Address represents a single row in contact_addresses.
// type = "tel"   -> target holds an E.164 phone number
// type = "email" -> target holds a lowercase email address
type Address struct {
	ID         uuid.UUID  `json:"id"`
	CustomerID uuid.UUID  `json:"customer_id"`
	ContactID  uuid.UUID  `json:"contact_id"`
	Type       string     `json:"type"`       // "tel" | "email"
	Target     string     `json:"target"`     // E.164 or email
	Name       string     `json:"name"`       // optional human-readable label
	Detail     string     `json:"detail"`     // optional free-form notes
	IsPrimary  bool       `json:"is_primary"`
	TMCreate   *time.Time `json:"tm_create"`
}

// Address type constants (discriminator values in contact_addresses.type column)
const (
	AddressTypeTel   = "tel"
	AddressTypeEmail = "email"
)

// AddressField represents a database/update field name for Address model
type AddressField string

// AddressField constants for use in AddressUpdate fields maps.
// Note: key names match the DB column names directly (no remapping needed).
const (
	AddressFieldTarget    AddressField = "target"     // maps to DB column target
	AddressFieldName      AddressField = "name"       // maps to DB column name
	AddressFieldDetail    AddressField = "detail"     // maps to DB column detail
	AddressFieldIsPrimary AddressField = "is_primary" // maps to DB column is_primary
)

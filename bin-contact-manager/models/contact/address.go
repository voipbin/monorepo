package contact

import (
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// Address represents a single row in contact_addresses.
// type = "tel"   -> target holds an E.164 phone number
// type = "email" -> target holds a lowercase email address
//
// Embeds commonaddress.Address (Type/Target/TargetName/Name/Detail) rather
// than hand-copying its fields -- this is the monorepo's standing convention
// for reusing a shared struct (see kase.Case embedding commonidentity.Owner).
type Address struct {
	commonaddress.Address
	ID         uuid.UUID  `json:"id"`
	CustomerID uuid.UUID  `json:"customer_id"`
	ContactID  uuid.UUID  `json:"contact_id"`
	IsPrimary  bool       `json:"is_primary"`
	TMCreate   *time.Time `json:"tm_create"`
}

// Address type constants. Reuse commonaddress.Type's canonical values --
// do not redeclare the string literals here. contact.Address intentionally
// accepts ONLY these two of commonaddress.Type's 10 possible values (see
// the explicit whitelist validation added to Create/AddAddress/UpdateAddress
// in pkg/contacthandler/contact.go as part of this same change).
const (
	AddressTypeTel   = commonaddress.TypeTel
	AddressTypeEmail = commonaddress.TypeEmail
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

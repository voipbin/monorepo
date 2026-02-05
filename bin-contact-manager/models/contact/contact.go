package contact

import (
	"fmt"
	"reflect"
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"

	"github.com/gofrs/uuid"
)

// Contact represents a CRM-style contact record for a customer.
// A contact stores information about a person or entity that the customer
// communicates with. Contacts can have multiple phone numbers and email
// addresses, and can be organized using tags.
//
// Contacts are tenant-isolated by CustomerID. Each customer maintains
// their own separate contact database.
type Contact struct {
	commonidentity.Identity // ID (contact UUID), CustomerID (tenant UUID)

	// -------------------------------------------------------------------------
	// Basic contact information
	// -------------------------------------------------------------------------

	// FirstName is the person's first/given name.
	// Example: "John"
	FirstName string `json:"first_name" db:"first_name"`

	// LastName is the person's last/family name.
	// Example: "Smith"
	LastName string `json:"last_name" db:"last_name"`

	// DisplayName is the preferred name to show in UIs and caller ID.
	// Can be a full name, nickname, or company name.
	// Examples: "John Smith", "Acme Corp", "Dr. Jane Doe"
	DisplayName string `json:"display_name" db:"display_name"`

	// Company is the organization or company the contact belongs to.
	// Example: "Acme Corporation"
	Company string `json:"company" db:"company"`

	// JobTitle is the contact's role or position within their organization.
	// Examples: "Sales Manager", "CEO", "Support Engineer"
	JobTitle string `json:"job_title" db:"job_title"`

	// -------------------------------------------------------------------------
	// Tracking and integration fields
	// -------------------------------------------------------------------------

	// Source indicates how this contact was created. Useful for analytics
	// and understanding contact origin.
	// Valid values:
	//   - "manual": Created via UI or direct API call by a user
	//   - "import": Bulk imported from CSV, Excel, or other file format
	//   - "api": Created programmatically via API integration
	//   - "sync": Automatically synced from an external CRM system
	Source string `json:"source" db:"source"`

	// ExternalID stores the unique identifier from an external system.
	// Used when contacts are imported or synced from third-party CRMs
	// (Salesforce, HubSpot, Zoho, etc.) or internal systems.
	//
	// Purpose:
	//   - Deduplication: Prevent creating duplicates during re-import
	//   - Two-way sync: Update the original record in the source system
	//   - Referential integrity: Maintain links to source system
	//
	// Examples:
	//   - "sf_003x000001ABC" (Salesforce contact ID)
	//   - "hubspot_12345" (HubSpot contact ID)
	//   - "erp_CUST-00001" (Internal ERP customer ID)
	ExternalID string `json:"external_id" db:"external_id"`

	// Notes stores free-form text notes about the contact.
	// Can contain any relevant information: call summaries, preferences,
	// special instructions, or general observations.
	// Supports multi-line text.
	Notes string `json:"notes" db:"notes"`

	// -------------------------------------------------------------------------
	// Related data
	// These fields are populated on read operations and can be included
	// on create. They are stored in separate database tables.
	// -------------------------------------------------------------------------

	// PhoneNumbers contains all phone numbers associated with this contact.
	// A contact can have multiple phone numbers (mobile, work, home, etc.).
	// Phone numbers support E.164 normalization for reliable lookup.
	PhoneNumbers []PhoneNumber `json:"phone_numbers,omitempty" db:"-"`

	// Emails contains all email addresses associated with this contact.
	// A contact can have multiple email addresses (work, personal, etc.).
	Emails []Email `json:"emails,omitempty" db:"-"`

	// TagIDs contains the IDs of tags assigned to this contact.
	// Tags are managed by bin-tag-manager and referenced here by ID.
	// Used for categorization, filtering, and organization.
	TagIDs []uuid.UUID `json:"tag_ids,omitempty" db:"-"`

	// -------------------------------------------------------------------------
	// Timestamps
	// -------------------------------------------------------------------------

	// TMCreate is when the contact was created.
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`

	// TMUpdate is when the contact was last modified.
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`

	// TMDelete is the soft-delete timestamp. If set, the contact is
	// considered deleted but data is retained for recovery/audit.
	// Nil means the contact is active.
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// Source type constants
const (
	SourceManual string = "manual" // Created via UI or direct API call
	SourceImport string = "import" // Bulk imported from file
	SourceAPI    string = "api"    // Created programmatically via API
	SourceSync   string = "sync"   // Synced from external CRM
)

// Matches returns true if the given items are the same.
// Ignores timestamp fields for comparison.
func (c *Contact) Matches(x interface{}) bool {
	comp, ok := x.(*Contact)
	if !ok {
		return false
	}
	a := *c

	// Ignore timestamp fields
	a.TMCreate = comp.TMCreate
	a.TMUpdate = comp.TMUpdate
	a.TMDelete = comp.TMDelete

	return reflect.DeepEqual(a, *comp)
}

// String returns a string representation of the contact
func (c *Contact) String() string {
	return fmt.Sprintf("%v", *c)
}

package customerdomain

import (
	"time"

	"github.com/gofrs/uuid"
)

// CustomerDomain maps a customer to its SIP domain label and full realm.
// One row per customer, created on customer_created (or lazily at the first
// extension creation) and hard-deleted on customer_deleted.
//
// Internal-only entity: no WebhookMessage — the domain is exposed to users
// only through Extension.DomainName.
type CustomerDomain struct {
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`

	DomainLabel string `json:"domain_label" db:"domain_label"` // 4-char base36 label (customer uuid string for backfilled legacy rows)
	Realm       string `json:"realm" db:"realm"`               // full realm. <domain_label>.<base domain name extension>

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
}

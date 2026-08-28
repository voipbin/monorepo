package accesskey

import (
	"time"

	"github.com/gofrs/uuid"
)

// Accesskey defines
type Accesskey struct {
	ID         uuid.UUID `json:"id" db:"id,uuid"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`

	Name   string `json:"name,omitempty" db:"name"`
	Detail string `json:"detail,omitempty" db:"detail"`

	// TokenHash is the SHA-256 hex digest of the token. Tagged json:"-" to prevent
	// exposure in API responses. Note: empty when loaded from cache (JSON serialization).
	TokenHash string `json:"-" db:"token_hash"`
	TokenPrefix string `json:"token_prefix" db:"token_prefix"`

	// RawToken holds the plain-text token temporarily during creation.
	// NOT stored in the database. Travels via RPC for one-time return to API.
	RawToken string `json:"raw_token,omitempty" db:"-"`

	TMExpire *time.Time `json:"tm_expire" db:"tm_expire"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// EventSubscriptionID returns the subscription address of this type on the global topic
// exchange `bin-manager.event`: the resource's own id (VOIP-1404 §4.2, VOIP-1419).
func (h *Accesskey) EventSubscriptionID() string {
	return h.ID.String()
}

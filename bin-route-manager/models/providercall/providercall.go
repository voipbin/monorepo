package providercall

import (
	"time"

	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// ProviderCall records an admin-triggered call placed through a specific provider.
// It captures the admin's request info (who, which provider, destinations, options)
// and the IDs of the calls/groupcalls created by the underlying `CallV1CallsCreate`
// invocation so the admin can correlate the test record back to the resulting call
// records without embedding them (atomic-API rule).
type ProviderCall struct {
	ID uuid.UUID `json:"id" db:"id,uuid"`

	// Requested
	CustomerID   uuid.UUID               `json:"customer_id"   db:"customer_id,uuid"`
	ProviderID   uuid.UUID               `json:"provider_id"   db:"provider_id,uuid"`
	FlowID       uuid.UUID               `json:"flow_id"       db:"flow_id,uuid"` // uuid.Nil when not provided
	Source       *commonaddress.Address  `json:"source"        db:"source,json"`
	Destinations []commonaddress.Address `json:"destinations"  db:"destinations,json"`
	Anonymous    string                  `json:"anonymous"     db:"anonymous"`

	// Created — IDs only, per the atomic-API rule (do not embed Call/Groupcall)
	CallIDs      []uuid.UUID `json:"call_ids"      db:"call_ids,json"`
	GroupcallIDs []uuid.UUID `json:"groupcall_ids" db:"groupcall_ids,json"`

	// Timestamps
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

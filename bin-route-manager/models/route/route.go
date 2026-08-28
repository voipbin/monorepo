package route

import (
	"time"

	cmcustomer "monorepo/bin-customer-manager/models/customer"

	"github.com/gofrs/uuid"
)

// Route defines
type Route struct {
	ID         uuid.UUID `json:"id" db:"id,uuid"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`

	Name   string `json:"name" db:"name"`
	Detail string `json:"detail" db:"detail"`

	ProviderID uuid.UUID `json:"provider_id" db:"provider_id,uuid"`
	Priority   int       `json:"priority" db:"priority"`

	Target string `json:"target" db:"target"` // country code or all

	// timestamp
	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

// EventSubscriptionID returns the subscription address of this type on the global topic
// exchange `bin-manager.event`: the resource's own id (VOIP-1404 §4.2, VOIP-1419).
func (h *Route) EventSubscriptionID() string {
	return h.ID.String()
}

// list of defined target
const (
	TargetAll = "all" // route target for all destination.
)

// CustomerIDBasicRoute is the customer ID for system-wide default routes.
// Deprecated: Use cmcustomer.IDBasicRoute from bin-customer-manager/models/customer instead.
// Kept as an alias for backward compatibility.
var CustomerIDBasicRoute = cmcustomer.IDBasicRoute

package route

import "github.com/gofrs/uuid"

// Route defines
type Route struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	ProviderID uuid.UUID `json:"provider_id"`
	Priority   int       `json:"priority"`

	Target string `json:"target"` // country code or all

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// list of defined target
const (
	TargetAll = "all" // route target for all destination.
)

// list of defined customer id
var (
	CustomerIDBasicRoute uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")
)

package route

import "github.com/gofrs/uuid"

// Route defines
type Route struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	ProviderID uuid.UUID `json:"provider_id"`
	Priority   int       `json:"priority"`

	Target string `json:"target"` // country code or all

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
}

// list of defined target
const (
	TargetAll = "all" // route target for all destination.
)

// list of defined customer id
var (
	CustomerIDDefault uuid.UUID = uuid.FromStringOrNil("efdcb8da-43cc-11ed-9b05-83b4ef0da730")
)

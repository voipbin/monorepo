package route

import (
	"time"

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

// list of defined target
const (
	TargetAll = "all" // route target for all destination.
)

// list of defined customer id
var (
	CustomerIDBasicRoute uuid.UUID = uuid.FromStringOrNil("00000000-0000-0000-0000-000000000001")
)

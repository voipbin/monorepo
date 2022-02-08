package number

import (
	"github.com/gofrs/uuid"
)

// Number struct represent order number information
type Number struct {
	ID     uuid.UUID `json:"id"`
	Number string    `json:"number"`
	FlowID uuid.UUID `json:"flow_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Status Status `json:"status"`

	T38Enabled       bool `json:"t38_enabled"`
	EmergencyEnabled bool `json:"emergency_enabled"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

// Status type
type Status string

// List of NumberStatus types
const (
	StatusActive  Status = "active"
	StatusDeleted Status = "deleted"
)

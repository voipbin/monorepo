package extension

import (
	"github.com/gofrs/uuid"
)

// Extension struct
// used only for the swag.
type Extension struct {
	ID uuid.UUID `json:"id"`

	Name     string    `json:"name"`
	Detail   string    `json:"detail"`
	DomainID uuid.UUID `json:"domain_id"`

	Extension string `json:"extension"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

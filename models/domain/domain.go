package domain

import (
	"github.com/gofrs/uuid"
)

// Domain struct
// used only for the swag.
type Domain struct {
	ID uuid.UUID `json:"id"`

	Name       string `json:"name"`
	Detail     string `json:"detail"`
	DomainName string `json:"domain_name"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

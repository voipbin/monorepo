package domain

import "github.com/gofrs/uuid"

// Domain struct
type Domain struct {
	ID     uuid.UUID `json:"id"`
	UserID uint64    `json:"user_id"`

	Name       string `json:"name"`
	Detail     string `json:"detail"`
	DomainName string `json:"domain_name"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

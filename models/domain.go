package models

import "github.com/gofrs/uuid"

// Domain struct
type Domain struct {
	ID     uuid.UUID `json:"id"`
	UserID int       `json:"user_id"`

	DomainName string `json:"domain_name"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

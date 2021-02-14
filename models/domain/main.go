package domain

import "github.com/gofrs/uuid"

// Domain struct for client show
type Domain struct {
	ID     uuid.UUID `json:"id"`
	UserID uint64    `json:"user_id"`

	DomainName string `json:"domain_name"`

	Name   string `json:"name"`   // Name
	Detail string `json:"detail"` // Detail

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.

}

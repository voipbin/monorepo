package domain

import "github.com/gofrs/uuid"

// Domain struct
type Domain struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name       string `json:"name"`
	Detail     string `json:"detail"`
	DomainName string `json:"domain_name"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

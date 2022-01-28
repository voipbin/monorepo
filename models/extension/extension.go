package extension

import "github.com/gofrs/uuid"

// Extension struct
type Extension struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name     string    `json:"name"`
	Detail   string    `json:"detail"`
	DomainID uuid.UUID `json:"domain_id"`

	// asterisk resources
	EndpointID string `json:"endpoint_id"`
	AORID      string `json:"aor_id"`
	AuthID     string `json:"auth_id"`

	Extension string `json:"extension"`
	Password  string `json:"password"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

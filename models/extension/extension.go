package extension

import "github.com/gofrs/uuid"

// Extension struct
type Extension struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	// asterisk resources
	EndpointID string `json:"endpoint_id"`
	AORID      string `json:"aor_id"`
	AuthID     string `json:"auth_id"`

	Extension string `json:"extension"`

	DomainName string `json:"domain_name"` // same as the CustomerID. This used by the kamailio's INVITE validation
	Username   string `json:"username"`    // same as the Extension. This used by the kamailio's INVITE validation
	Password   string `json:"password"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

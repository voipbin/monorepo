package accesskey

import "github.com/gofrs/uuid"

// Accesskey defines
type Accesskey struct {
	ID         uuid.UUID `json:"id"`
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`

	Token string `json:"token"`

	TMExpire string `json:"tm_expire,omitempty"`

	TMCreate string `json:"tm_create,omitempty"`
	TMUpdate string `json:"tm_update,omitempty"`
	TMDelete string `json:"tm_delete,omitempty"`
}

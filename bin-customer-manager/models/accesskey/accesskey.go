package accesskey

import "github.com/gofrs/uuid"

// Accesskey defines
type Accesskey struct {
	ID         uuid.UUID `json:"id" db:"id,uuid"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`

	Name   string `json:"name,omitempty" db:"name"`
	Detail string `json:"detail,omitempty" db:"detail"`

	Token string `json:"token" db:"token"`

	TMExpire string `json:"tm_expire,omitempty" db:"tm_expire"`

	TMCreate string `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate string `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete string `json:"tm_delete,omitempty" db:"tm_delete"`
}

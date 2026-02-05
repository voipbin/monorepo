package accesskey

import (
	"time"

	"github.com/gofrs/uuid"
)

// Accesskey defines
type Accesskey struct {
	ID         uuid.UUID `json:"id" db:"id,uuid"`
	CustomerID uuid.UUID `json:"customer_id" db:"customer_id,uuid"`

	Name   string `json:"name,omitempty" db:"name"`
	Detail string `json:"detail,omitempty" db:"detail"`

	Token string `json:"token" db:"token"`

	TMExpire *time.Time `json:"tm_expire" db:"tm_expire"`

	TMCreate *time.Time `json:"tm_create" db:"tm_create"`
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"`
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"`
}

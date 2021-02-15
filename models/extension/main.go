package extension

import "github.com/gofrs/uuid"

// Extension struct
type Extension struct {
	ID     uuid.UUID `json:"id"`
	UserID uint64    `json:"user_id"`

	Name     string    `json:"name"`
	Detail   string    `json:"detail"`
	DomainID uuid.UUID `json:"domain_id"`

	Extension string `json:"extension"`
	Password  string `json:"password"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

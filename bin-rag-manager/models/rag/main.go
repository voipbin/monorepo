package rag

import (
	"time"

	"github.com/gofrs/uuid"
)

// Rag represents a knowledge base container
type Rag struct {
	ID          uuid.UUID  `json:"id,omitempty" db:"id,uuid"`
	CustomerID  uuid.UUID  `json:"customer_id,omitempty" db:"customer_id,uuid"`
	Name        string     `json:"name,omitempty" db:"name"`
	Description string     `json:"description,omitempty" db:"description"`
	TMCreate    *time.Time `json:"tm_create,omitempty" db:"tm_create"`
	TMUpdate    *time.Time `json:"tm_update,omitempty" db:"tm_update"`
	TMDelete    *time.Time `json:"tm_delete,omitempty" db:"tm_delete"`
}

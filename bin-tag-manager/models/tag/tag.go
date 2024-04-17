package tag

import "github.com/gofrs/uuid"

// Tag data model
type Tag struct {
	ID         uuid.UUID `json:"id"`          // tag id
	CustomerID uuid.UUID `json:"customer_id"` // owner's id

	Name   string `json:"name"`   // tag's name
	Detail string `json:"detail"` // tag's detail

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

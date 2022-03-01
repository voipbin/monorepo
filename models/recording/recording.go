package recording

import (
	"github.com/gofrs/uuid"
)

// Recording struct represent record information
// used only for the swag.
type Recording struct {
	ID          uuid.UUID `json:"id"`
	Type        string    `json:"type"`
	ReferenceID uuid.UUID `json:"reference_id"`
	Status      string    `json:"status"`
	Format      string    `json:"format"`

	TMStart string `json:"tm_start"`
	TMEnd   string `json:"tm_end"`

	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

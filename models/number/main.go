package number

import "github.com/gofrs/uuid"

// Number struct represent number information
type Number struct {
	ID     uuid.UUID
	Number string
	FlowID uuid.UUID
	UserID uint64

	// timestamp
	TMCreate string `json:"tm_create"`
	TMUpdate string `json:"tm_update"`
	TMDelete string `json:"tm_delete"`
}

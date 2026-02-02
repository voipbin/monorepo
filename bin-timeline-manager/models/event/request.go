package event

import (
	"github.com/gofrs/uuid"
)

// EventListRequest represents the request for listing events.
type EventListRequest struct {
	Publisher string    `json:"publisher"`
	ID        uuid.UUID `json:"id"`
	Events    []string  `json:"events"`

	// Pagination
	PageToken string `json:"page_token,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}

// Default and max page sizes
const (
	DefaultPageSize = 100
	MaxPageSize     = 1000
)

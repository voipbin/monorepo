package models

import (
	"github.com/gofrs/uuid"
)

// Flow struct for client show
type Flow struct {
	ID     uuid.UUID `json:"id"` // Flow's ID
	UserID uint64    `json:"-"`  // Flow owner's User ID

	Name   string `json:"name"`   // Name
	Detail string `json:"detail"` // Detail

	Actions []Action `json:"actions"` // Actions

	Persist    bool   `json:"-"` // Persist
	WebhookURI string `json:"webhook_uri"`

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

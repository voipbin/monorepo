package flow

import (
	"github.com/gofrs/uuid"

	"gitlab.com/voipbin/bin-manager/api-manager.git/models/action"
)

// Flow struct for client show
// used only for the swag.
type Flow struct {
	ID         uuid.UUID `json:"id"` // Flow's ID
	CustomerID uuid.UUID `json:"-"`  // Flow owner's customer ID

	Name   string `json:"name"`   // Name
	Detail string `json:"detail"` // Detail

	Actions []action.Action `json:"actions"` // Actions

	Persist    bool   `json:"-"` // Persist
	WebhookURI string `json:"webhook_uri"`

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

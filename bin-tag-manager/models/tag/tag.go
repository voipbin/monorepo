package tag

import (
	"time"

	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Tag data model
type Tag struct {
	commonidentity.Identity

	Name   string `json:"name" db:"name"`     // tag's name
	Detail string `json:"detail" db:"detail"` // tag's detail

	TMCreate *time.Time `json:"tm_create" db:"tm_create"` // Created timestamp.
	TMUpdate *time.Time `json:"tm_update" db:"tm_update"` // Updated timestamp.
	TMDelete *time.Time `json:"tm_delete" db:"tm_delete"` // Deleted timestamp.
}

// EventSubscriptionID returns the subscription address of this type on the global topic
// exchange `bin-manager.event`: the resource's own id (VOIP-1404 §4.2, VOIP-1419).
func (h *Tag) EventSubscriptionID() string {
	return h.ID.String()
}

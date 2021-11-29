package tag

import (
	"github.com/gofrs/uuid"
	amtag "gitlab.com/voipbin/bin-manager/agent-manager.git/models/tag"
)

// Tag data model
type Tag struct {
	ID     uuid.UUID `json:"id"`      // tag id
	UserID uint64    `json:"user_id"` // owned user's id

	Name   string `json:"name"`   // tag's name
	Detail string `json:"detail"` // tag's detail

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

// ConvertToTag define
func ConvertToTag(t *amtag.Tag) *Tag {
	return &Tag{
		ID:     t.ID,
		UserID: t.UserID,

		Name:   t.Name,
		Detail: t.Detail,

		TMCreate: t.TMCreate,
		TMUpdate: t.TMUpdate,
		TMDelete: t.TMDelete,
	}
}

// ConvertToAMTag define
func ConvertToAMTag(t *Tag) *amtag.Tag {
	return &amtag.Tag{
		ID:     t.ID,
		UserID: t.UserID,

		Name:   t.Name,
		Detail: t.Detail,

		TMCreate: t.TMCreate,
		TMUpdate: t.TMUpdate,
		TMDelete: t.TMDelete,
	}
}

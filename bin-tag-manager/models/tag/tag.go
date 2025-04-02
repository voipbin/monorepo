package tag

import (
	commonidentity "monorepo/bin-common-handler/models/identity"
)

// Tag data model
type Tag struct {
	commonidentity.Identity

	Name   string `json:"name"`   // tag's name
	Detail string `json:"detail"` // tag's detail

	TMCreate string `json:"tm_create"` // Created timestamp.
	TMUpdate string `json:"tm_update"` // Updated timestamp.
	TMDelete string `json:"tm_delete"` // Deleted timestamp.
}

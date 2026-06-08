package request

import (
	"github.com/gofrs/uuid"

	commonoutline "monorepo/bin-common-handler/models/outline"
)

// V1DataEventsPost represents the request for listing events.
type V1DataEventsPost struct {
	Publisher  commonoutline.ServiceName `json:"publisher"`
	ResourceID uuid.UUID                 `json:"resource_id"`
	Events     []string                  `json:"events"`

	// Pagination
	PageToken string `json:"page_token,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}

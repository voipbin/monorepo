package event

import (
	"github.com/gofrs/uuid"

	commonoutline "monorepo/bin-common-handler/models/outline"
)

// EventListRequest represents the request for listing events.
// Used by the request handler to communicate with timeline-manager.
type EventListRequest struct {
	Publisher  commonoutline.ServiceName `json:"publisher"`
	ResourceID uuid.UUID                 `json:"resource_id"`
	Events     []string                  `json:"events"`

	// Pagination
	PageToken string `json:"page_token,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}

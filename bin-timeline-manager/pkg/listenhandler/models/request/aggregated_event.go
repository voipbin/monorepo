package request

import (
	"github.com/gofrs/uuid"
)

// V1DataAggregatedEventsPost represents the request for listing aggregated events.
type V1DataAggregatedEventsPost struct {
	ActiveflowID uuid.UUID `json:"activeflow_id"`

	// Pagination
	PageToken string `json:"page_token,omitempty"`
	PageSize  int    `json:"page_size,omitempty"`
}

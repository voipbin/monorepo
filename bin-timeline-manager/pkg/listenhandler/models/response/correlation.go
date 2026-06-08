package response

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-timeline-manager/models/correlation"
)

// V1DataResourceCorrelationGet represents the correlation graph for a resource.
type V1DataResourceCorrelationGet struct {
	ResourceID    uuid.UUID                     `json:"resource_id"`
	ResourceFound bool                          `json:"resource_found"`
	ActiveflowID  uuid.UUID                     `json:"activeflow_id"`
	Truncated     bool                          `json:"truncated"`
	Resources     []*correlation.PublisherGroup `json:"resources"`
}

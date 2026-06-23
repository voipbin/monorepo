package request

import (
	"github.com/gofrs/uuid"
)

// V1DataAnalysesPost represents the trigger request.
// customer_id is the ownership authority (server-injected upstream by api-manager).
type V1DataAnalysesPost struct {
	CustomerID   uuid.UUID `json:"customer_id"`
	ActiveflowID uuid.UUID `json:"activeflow_id"`
	Reanalyze    bool      `json:"reanalyze,omitempty"`
}

// V1DataAnalysesGet represents the get/list query params carried in the body.
type V1DataAnalysesGet struct {
	CustomerID   uuid.UUID `json:"customer_id"`
	ActiveflowID uuid.UUID `json:"activeflow_id,omitempty"`
	Status       string    `json:"status,omitempty"`

	// Pagination
	PageToken string `json:"page_token,omitempty"`
	PageSize  uint64 `json:"page_size,omitempty"`
}

// V1DataAnalysesIDDelete represents the delete request (ownership-checked).
type V1DataAnalysesIDDelete struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

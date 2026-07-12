package request

import "github.com/gofrs/uuid"

// V1DataCasesIDResolutionsPost is the request body for
// POST /v1/cases/{id}/resolutions.
type V1DataCasesIDResolutionsPost struct {
	CustomerID     uuid.UUID `json:"customer_id"`
	ContactID      uuid.UUID `json:"contact_id"`
	ResolutionType string    `json:"resolution_type"`
	ResolvedByType string    `json:"resolved_by_type"`
	ResolvedByID   uuid.UUID `json:"resolved_by_id"`
}

// V1DataCasesIDResolutionsIDDelete is the request body for
// DELETE /v1/cases/{id}/resolutions/{resolution_id}.
type V1DataCasesIDResolutionsIDDelete struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

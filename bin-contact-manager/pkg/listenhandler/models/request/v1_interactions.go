package request

import "github.com/gofrs/uuid"

// V1DataInteractionsResolutionsPost is the request body for
// POST /v1/interactions/{id}/resolutions.
type V1DataInteractionsResolutionsPost struct {
	CustomerID     uuid.UUID `json:"customer_id"`
	ContactID      uuid.UUID `json:"contact_id"`
	ResolutionType string    `json:"resolution_type"`
	ResolvedByType string    `json:"resolved_by_type"`
	ResolvedByID   uuid.UUID `json:"resolved_by_id"`
}

// V1DataInteractionsResolutionsIDDelete is the request body for
// DELETE /v1/interactions/{id}/resolutions/{rid}.
// VoIPBin DELETE handlers carry customerID in the request body.
type V1DataInteractionsResolutionsIDDelete struct {
	CustomerID uuid.UUID `json:"customer_id"`
}

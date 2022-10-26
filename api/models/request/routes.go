package request

import (
	"github.com/gofrs/uuid"
)

// ParamRoutesGET is request param define for GET /routes
type ParamRoutesGET struct {
	CustomerID string `form:"customer_id"`
	Pagination
}

// BodyRoutesPOST is request body define for POST /routes
type BodyRoutesPOST struct {
	CustomerID uuid.UUID `json:"customer_id"`
	ProviderID uuid.UUID `json:"provider_id"`
	Priority   int       `json:"priority"`
	Target     string    `json:"target"`
}

// BodyRoutesIDPUT is request body define for PUT /routes/<route-id>
type BodyRoutesIDPUT struct {
	ProviderID uuid.UUID `json:"provider_id"`
	Priority   int       `json:"priority"`
	Target     string    `json:"target"`
}

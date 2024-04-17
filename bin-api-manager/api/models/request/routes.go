package request

import (
	"github.com/gofrs/uuid"
)

// ParamRoutesGET is request param define for
// GET /v1.0/routes
type ParamRoutesGET struct {
	CustomerID string `form:"customer_id"`
	Pagination
}

// BodyRoutesPOST is request body define for
// POST /v1.0/routes
type BodyRoutesPOST struct {
	CustomerID uuid.UUID `json:"customer_id"`
	Name       string    `json:"name"`
	Detail     string    `json:"detail"`
	ProviderID uuid.UUID `json:"provider_id"`
	Priority   int       `json:"priority"`
	Target     string    `json:"target"`
}

// BodyRoutesIDPUT is request body define for
// PUT /v1.0/routes/<route-id>
type BodyRoutesIDPUT struct {
	Name       string    `json:"name"`
	Detail     string    `json:"detail"`
	ProviderID uuid.UUID `json:"provider_id"`
	Priority   int       `json:"priority"`
	Target     string    `json:"target"`
}

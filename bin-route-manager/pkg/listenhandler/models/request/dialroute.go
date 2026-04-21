package request

import (
	"github.com/gofrs/uuid"

	rmroute "monorepo/bin-route-manager/models/route"
)

// V1DataDialroutesGet is
// v1 data type request struct for
// /v1/dialroutes GET
//
// Legacy top-level CustomerID/Target fields are retained alongside the
// Filters map for rolling-deploy backward compatibility: new clients populate
// both, old clients populate only CustomerID/Target, old servers read only
// CustomerID/Target, new servers prefer Filters when present. This lets
// bin-call-manager and bin-route-manager roll independently without
// silently mis-routing outgoing calls during the version-skew window.
type V1DataDialroutesGet struct {
	// Legacy flat fields (kept for backward compat during rolling deploy).
	CustomerID uuid.UUID `json:"customer_id,omitempty"`
	Target     string    `json:"target,omitempty"`

	// Newer structured fields.
	Filters           map[rmroute.Field]any `json:"filters,omitempty"`
	TargetProviderIDs []uuid.UUID           `json:"target_provider_ids,omitempty"`
}

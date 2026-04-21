package request

import (
	"github.com/gofrs/uuid"

	rmroute "monorepo/bin-route-manager/models/route"
)

// V1DataDialroutesGet is
// v1 data type request struct for
// /v1/dialroutes GET
type V1DataDialroutesGet struct {
	Filters           map[rmroute.Field]any `json:"filters,omitempty"`
	TargetProviderIDs []uuid.UUID           `json:"target_provider_ids,omitempty"`
}

package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"
	fmaction "monorepo/bin-flow-manager/models/action"

	"github.com/gofrs/uuid"
)

// V1DataProviderCallsPost is the request body for POST /v1/providercalls.
// bin-api-manager validates permission + provider_id then forwards this payload
// to bin-route-manager, which does the full orchestration: optional temp-flow
// creation, metadata construction (route_provider_ids + skip_source_validation),
// CallV1CallsCreate, and ProviderCall persistence with the resulting call IDs.
type V1DataProviderCallsPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	ProviderID uuid.UUID `json:"provider_id"`

	// Either FlowID or Actions (not both). When Actions is non-empty and
	// FlowID is uuid.Nil, route-manager creates a temporary flow for the call.
	FlowID  uuid.UUID        `json:"flow_id"`
	Actions []fmaction.Action `json:"actions"`

	Source       *commonaddress.Address  `json:"source"`
	Destinations []commonaddress.Address `json:"destinations"`
	Anonymous    string                  `json:"anonymous"`
}

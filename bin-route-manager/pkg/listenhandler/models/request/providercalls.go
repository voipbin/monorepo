package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// V1DataProviderCallsPost is the request body for POST /v1/providercalls.
// The caller (bin-api-manager's provider-call service handler) has already
// performed the call creation via CallV1CallsCreate and passes the resulting
// call_ids/groupcall_ids alongside the original request info.
type V1DataProviderCallsPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	ProviderID uuid.UUID `json:"provider_id"`
	FlowID     uuid.UUID `json:"flow_id"`

	Source       *commonaddress.Address  `json:"source"`
	Destinations []commonaddress.Address `json:"destinations"`
	Anonymous    string                  `json:"anonymous"`

	CallIDs      []uuid.UUID `json:"call_ids"`
	GroupcallIDs []uuid.UUID `json:"groupcall_ids"`
}

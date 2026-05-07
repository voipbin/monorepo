package request

import (
	"github.com/gofrs/uuid"
)

// V1DataOutboundConfigsPost is the request body for POST /v1/outbound_configs.
//
// Pointer fields preserve the "absent" (nil) vs. "explicit empty" (non-nil pointer
// to zero value) distinction used by outboundconfig.UpdateRequest in the partial-update
// SQL builder.
type V1DataOutboundConfigsPost struct {
	CustomerID           uuid.UUID `json:"customer_id"`
	Name                 *string   `json:"name,omitempty"`
	Detail               *string   `json:"detail,omitempty"`
	DestinationWhitelist *[]string `json:"destination_whitelist,omitempty"`
	Codecs               *string   `json:"codecs,omitempty"`
}

// V1DataOutboundConfigsIDPut is the request body for PUT /v1/outbound_configs/<id>.
type V1DataOutboundConfigsIDPut struct {
	Name                 *string   `json:"name,omitempty"`
	Detail               *string   `json:"detail,omitempty"`
	DestinationWhitelist *[]string `json:"destination_whitelist,omitempty"`
	Codecs               *string   `json:"codecs,omitempty"`
}

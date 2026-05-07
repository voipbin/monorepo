package request

import (
	outboundconfig "monorepo/bin-call-manager/models/outboundconfig"
	"github.com/gofrs/uuid"
)

// V1DataOutboundConfigsPost is the request body for POST /v1/outbound_configs.
type V1DataOutboundConfigsPost struct {
	CustomerID uuid.UUID                     `json:"customer_id"`
	Request    outboundconfig.UpdateRequest  `json:"request"`
}

// V1DataOutboundConfigsIDPut is the request body for PUT /v1/outbound_configs/<id>.
type V1DataOutboundConfigsIDPut struct {
	Request outboundconfig.UpdateRequest `json:"request"`
}

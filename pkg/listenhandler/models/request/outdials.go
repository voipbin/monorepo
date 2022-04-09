package request

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
)

// V1DataOutdialsPost is
// v1 data type request struct for
// /v1/outdials POST
type V1DataOutdialsPost struct {
	CustomerID uuid.UUID `json:"customer_id"` // flow's owner
	CampaignID uuid.UUID `json:"campaign_id"`

	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail
	Data   string `json:"data"`
}

// V1DataOutdialsIDTargetsPost is
// v1 data type request struct for
// /v1/outdials/<outdial-id>/targets POST
type V1DataOutdialsIDTargetsPost struct {
	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail
	Data   string `json:"data"`

	Destination0 *cmaddress.Address `json:"destination_0,omitempty"`
	Destination1 *cmaddress.Address `json:"destination_1,omitempty"`
	Destination2 *cmaddress.Address `json:"destination_2,omitempty"`
	Destination3 *cmaddress.Address `json:"destination_3,omitempty"`
	Destination4 *cmaddress.Address `json:"destination_4,omitempty"`
}

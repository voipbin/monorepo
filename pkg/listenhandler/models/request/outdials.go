package request

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"
)

// V1DataOutdialsPost is
// v1 data type request struct for
// /v1/outdials POST
type V1DataOutdialsPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	CampaignID uuid.UUID `json:"campaign_id"`

	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail
	Data   string `json:"data"`
}

// V1DataOutdialsIDPut is
// v1 data type request struct for
// /v1/outdials/<outdial-id> PUT
type V1DataOutdialsIDPut struct {
	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail
}

// V1DataOutdialsIDDataPut is
// v1 data type request struct for
// /v1/outdials/<outdial-id>/data PUT
type V1DataOutdialsIDDataPut struct {
	Data string `json:"data"`
}

// V1DataOutdialsIDCampaignIDPut is
// v1 data type request struct for
// /v1/outdials/<outdial-id>/campaign_id PUT
type V1DataOutdialsIDCampaignIDPut struct {
	CampaignID uuid.UUID `json:"campaign_id"`
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

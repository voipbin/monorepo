package request

import (
	"github.com/gofrs/uuid"
	cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

	"gitlab.com/voipbin/bin-manager/outdial-manager.git/models/outdialtarget"
)

// V1DataOutdialtargetsPost is
// v1 data type request struct for
// /v1/outdialtargets POST
type V1DataOutdialtargetsPost struct {
	OutdialID uuid.UUID `json:"outdial_id"`

	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail
	Data   string `json:"data"`

	Destination0 *cmaddress.Address `json:"destination_0,omitempty"`
	Destination1 *cmaddress.Address `json:"destination_1,omitempty"`
	Destination2 *cmaddress.Address `json:"destination_2,omitempty"`
	Destination3 *cmaddress.Address `json:"destination_3,omitempty"`
	Destination4 *cmaddress.Address `json:"destination_4,omitempty"`
}

// V1DataOutdialtargetsIDProgressingPost is
// v1 data type request struct for
// /v1/outdialtargets/<outdialtarget-id> PUT
type V1DataOutdialtargetsIDProgressingPost struct {
	DestinationIndex int `json:"destination_index"`
}

// V1DataOutdialtargetsIDStatusPut is
// v1 data type request struct for
// /v1/outdialtargets/<outdialtarget-id>/status PUT
type V1DataOutdialtargetsIDStatusPut struct {
	Status outdialtarget.Status `json:"status"`
}

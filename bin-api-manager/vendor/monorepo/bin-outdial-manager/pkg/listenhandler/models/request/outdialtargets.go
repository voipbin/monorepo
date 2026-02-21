package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"

	"monorepo/bin-outdial-manager/models/outdialtarget"
)

// V1DataOutdialtargetsPost is
// v1 data type request struct for
// /v1/outdialtargets POST
type V1DataOutdialtargetsPost struct {
	OutdialID uuid.UUID `json:"outdial_id"`

	Name   string `json:"name"`   // name
	Detail string `json:"detail"` // detail
	Data   string `json:"data"`

	Destination0 *commonaddress.Address `json:"destination_0,omitempty"`
	Destination1 *commonaddress.Address `json:"destination_1,omitempty"`
	Destination2 *commonaddress.Address `json:"destination_2,omitempty"`
	Destination3 *commonaddress.Address `json:"destination_3,omitempty"`
	Destination4 *commonaddress.Address `json:"destination_4,omitempty"`
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

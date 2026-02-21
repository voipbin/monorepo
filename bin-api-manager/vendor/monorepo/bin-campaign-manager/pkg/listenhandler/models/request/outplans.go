package request

import (
	commonaddress "monorepo/bin-common-handler/models/address"

	"github.com/gofrs/uuid"
)

// V1DataOutplansPost is
// v1 data type request struct for
// /v1/outplans POST
type V1DataOutplansPost struct {
	CustomerID uuid.UUID `json:"customer_id"`

	Name   string `json:"name"`
	Detail string `json:"detail"`

	Source *commonaddress.Address `json:"source"`

	DialTimeout int `json:"dial_timeout"`
	TryInterval int `json:"try_interval"`

	MaxTryCount0 int `json:"max_try_count_0"`
	MaxTryCount1 int `json:"max_try_count_1"`
	MaxTryCount2 int `json:"max_try_count_2"`
	MaxTryCount3 int `json:"max_try_count_3"`
	MaxTryCount4 int `json:"max_try_count_4"`
}

// V1DataOutplansIDPut is
// v1 data type request struct for
// /v1/outplans/<outplan_id> PUT
type V1DataOutplansIDPut struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// V1DataOutplansIDDialsPut is
// v1 data type request struct for
// /v1/outplans/<outplan_id>/dials PUT
type V1DataOutplansIDDialsPut struct {
	Source *commonaddress.Address `json:"source"`

	DialTimeout int `json:"dial_timeout"`
	TryInterval int `json:"try_interval"`

	MaxTryCount0 int `json:"max_try_count_0"`
	MaxTryCount1 int `json:"max_try_count_1"`
	MaxTryCount2 int `json:"max_try_count_2"`
	MaxTryCount3 int `json:"max_try_count_3"`
	MaxTryCount4 int `json:"max_try_count_4"`
}

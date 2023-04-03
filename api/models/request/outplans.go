package request

import commonaddress "gitlab.com/voipbin/bin-manager/common-handler.git/models/address"

// BodyOutplansPOST is rquest body define for
// POST /v1.0/outplans
type BodyOutplansPOST struct {
	Name         string                 `json:"name"`
	Detail       string                 `json:"detail"`
	Source       *commonaddress.Address `json:"source"`
	DialTimeout  int                    `json:"dial_timeout"`
	TryInterval  int                    `json:"try_interval"`
	MaxTryCount0 int                    `json:"max_try_count_0"`
	MaxTryCount1 int                    `json:"max_try_count_1"`
	MaxTryCount2 int                    `json:"max_try_count_2"`
	MaxTryCount3 int                    `json:"max_try_count_3"`
	MaxTryCount4 int                    `json:"max_try_count_4"`
}

// ParamOutplansGET is rquest param define for
// GET /v1.0/outplans
type ParamOutplansGET struct {
	Pagination
}

// BodyOutplansIDPUT is rquest body define for
// PUT /v1.0/outplans/<outplan-id>
type BodyOutplansIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// BodyOutplansIDDialInfoPUT is rquest body define for
// PUT /v1.0/outplans/<outplan-id>/dial_info
type BodyOutplansIDDialInfoPUT struct {
	Source       *commonaddress.Address `json:"source"`
	DialTimeout  int                    `json:"dial_timeout"`
	TryInterval  int                    `json:"try_interval"`
	MaxTryCount0 int                    `json:"max_try_count_0"`
	MaxTryCount1 int                    `json:"max_try_count_1"`
	MaxTryCount2 int                    `json:"max_try_count_2"`
	MaxTryCount3 int                    `json:"max_try_count_3"`
	MaxTryCount4 int                    `json:"max_try_count_4"`
}

package request

import cmaddress "gitlab.com/voipbin/bin-manager/call-manager.git/models/address"

// BodyOutplansPOST is rquest body define for POST /outplans
type BodyOutplansPOST struct {
	Name         string             `json:"name"`
	Detail       string             `json:"detail"`
	Source       *cmaddress.Address `json:"source"`
	DialTimeout  int                `json:"dial_timeout"`
	TryInterval  int                `json:"try_interval"`
	MaxTryCount0 int                `json:"max_try_count_0"`
	MaxTryCount1 int                `json:"max_try_count_1"`
	MaxTryCount2 int                `json:"max_try_count_2"`
	MaxTryCount3 int                `json:"max_try_count_3"`
	MaxTryCount4 int                `json:"max_try_count_4"`
}

// ParamOutplansGET is rquest param define for GET /outplans
type ParamOutplansGET struct {
	Pagination
}

// BodyOutplansIDPUT is rquest body define for PUT /outplans/{id}
type BodyOutplansIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// BodyOutplansIDDialInfoPUT is rquest body define for PUT /outplans/{id}/dial_info
type BodyOutplansIDDialInfoPUT struct {
	Source       *cmaddress.Address `json:"source"`
	DialTimeout  int                `json:"dial_timeout"`
	TryInterval  int                `json:"try_interval"`
	MaxTryCount0 int                `json:"max_try_count_0"`
	MaxTryCount1 int                `json:"max_try_count_1"`
	MaxTryCount2 int                `json:"max_try_count_2"`
	MaxTryCount3 int                `json:"max_try_count_3"`
	MaxTryCount4 int                `json:"max_try_count_4"`
}

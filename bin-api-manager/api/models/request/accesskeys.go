package request

// ParamAccesskeysGET is rquest param define for
// GET /v1.0/accesskeys
type ParamAccesskeysGET struct {
	Pagination
}

// BodyAccesskeysPOST is rquest body define for
// POST /v1.0/accesskeys
type BodyAccesskeysPOST struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
	Expire int32  `json:"expire"`
}

// BodyAccesskeysIDPUT is rquest body define for
// PUT /v1.0/accesskeys/<accesskey-id>
type BodyAccesskeysIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

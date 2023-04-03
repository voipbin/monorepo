package request

// ParamTagsGET is rquest param define for
// GET /v1.0/tags
type ParamTagsGET struct {
	Pagination
}

// BodyTagsPOST is rquest body define for
// POST /v1.0/tags
type BodyTagsPOST struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// BodyTagsIDPUT is rquest body define for
// PUT /v1.0/tags/<tag-id>
type BodyTagsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

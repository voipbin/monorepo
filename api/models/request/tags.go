package request

// ParamTagsGET is rquest param define for GET /tags
type ParamTagsGET struct {
	Pagination
}

// BodyTagsPOST is rquest body define for POST /tags
type BodyTagsPOST struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// BodyTagsIDPUT is rquest body define for PUT /tags/<tag-id>
type BodyTagsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

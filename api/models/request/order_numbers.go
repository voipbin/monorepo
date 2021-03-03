package request

// ParamNumbersGET is request param define for GET /numbers
type ParamNumbersGET struct {
	Pagination
}

// BodyOrderNumbersPOST is request param define for POST /numbers
type BodyOrderNumbersPOST struct {
	Number string `json:"number"`
}

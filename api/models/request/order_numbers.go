package request

// ParamOrderNumbersGET is request param define for GET /order_numbers
type ParamOrderNumbersGET struct {
	Pagination
}

// BodyOrderNumbersPOST is request param define for POST /order_numbers
type BodyOrderNumbersPOST struct {
	Number string `json:"number"`
}

package request

import "github.com/gofrs/uuid"

// ParamNumbersGET is request param define for GET /numbers
type ParamNumbersGET struct {
	Pagination
}

// BodyNumbersPOST is request param define for POST /numbers
type BodyNumbersPOST struct {
	Number string `json:"number"`
}

// BodyNumbersIDPUT is request param define for PUT /numbers/<id>
type BodyNumbersIDPUT struct {
	FlowID uuid.UUID `json:"flow_id"`
}

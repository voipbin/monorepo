package request

import "github.com/gofrs/uuid"

// ParamNumbersGET is request param define for GET /numbers
type ParamNumbersGET struct {
	Pagination
}

// BodyNumbersPOST is request param define for POST /numbers
type BodyNumbersPOST struct {
	Number        string    `json:"number"`
	CallFlowID    uuid.UUID `json:"call_flow_id"`
	MessageFlowID uuid.UUID `json:"message_flow_id"`
	Name          string    `json:"name"`
	Detail        string    `json:"detail"`
}

// BodyNumbersIDPUT is request param define for PUT /numbers/<id>
type BodyNumbersIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// BodyNumbersIDFlowIDPUT is request param define for PUT /numbers/<id>/flow_id
type BodyNumbersIDFlowIDPUT struct {
	CallFlowID    uuid.UUID `json:"call_flow_id"`
	MessageFlowID uuid.UUID `json:"message_flow_id"`
}

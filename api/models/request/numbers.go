package request

import "github.com/gofrs/uuid"

// ParamNumbersGET is request param define for
// GET /v1.0/numbers
type ParamNumbersGET struct {
	Pagination
}

// BodyNumbersPOST is request param define for
// POST /v1.0/numbers
type BodyNumbersPOST struct {
	Number        string    `json:"number"`
	CallFlowID    uuid.UUID `json:"call_flow_id"`
	MessageFlowID uuid.UUID `json:"message_flow_id"`
	Name          string    `json:"name"`
	Detail        string    `json:"detail"`
}

// BodyNumbersIDPUT is request param define for
// PUT /v1.0/numbers/<number-id>
type BodyNumbersIDPUT struct {
	CallFlowID    uuid.UUID `json:"call_flow_id"`
	MessageFlowID uuid.UUID `json:"message_flow_id"`
	Name          string    `json:"name"`
	Detail        string    `json:"detail"`
}

// BodyNumbersIDFlowIDPUT is request param define for
// PUT /v1.0/numbers/<number-id>/flow_id
type BodyNumbersIDFlowIDPUT struct {
	CallFlowID    uuid.UUID `json:"call_flow_id"`
	MessageFlowID uuid.UUID `json:"message_flow_id"`
}

// BodyNumbersRenewPOST is rquest body define for
// POST /v1.0/numbers/renew
type BodyNumbersRenewPOST struct {
	TMRenew string `json:"tm_renew"`
}

package request

import "github.com/gofrs/uuid"

// V1DataNumbersPost is
// v1 data type request struct for
// /v1/numbers POST
type V1DataNumbersPost struct {
	CustomerID uuid.UUID `json:"customer_id"`
	FlowID     uuid.UUID `json:"flow_id"`
	Number     string    `json:"number"`
	Name       string    `json:"name"`
	Detail     string    `json:"detail"`
}

// V1DataNumbersIDPut is
// v1 data type request struct for
// /v1/numbers/<id> PUT
type V1DataNumbersIDPut struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// V1DataNumbersIDFlowIDPut is
// v1 data type request struct for
// /v1/numbers/<id>/flow_id PUT
type V1DataNumbersIDFlowIDPut struct {
	FlowID uuid.UUID `json:"flow_id"`
}

package request

import (
	"github.com/gofrs/uuid"

	"monorepo/bin-number-manager/models/number"
)

// V1DataNumbersPost is
// v1 data type request struct for
// /v1/numbers POST
type V1DataNumbersPost struct {
	CustomerID    uuid.UUID   `json:"customer_id"`
	Number        string      `json:"number"`
	Type          number.Type `json:"type"`
	CallFlowID    uuid.UUID   `json:"call_flow_id"`
	MessageFlowID uuid.UUID   `json:"message_flow_id"`
	Name          string      `json:"name"`
	Detail        string      `json:"detail"`
}

// V1DataNumbersRenewPost is
// v1 data type request struct for
// /v1/numbers/renew POST
type V1DataNumbersRenewPost struct {
	TMRenew string `json:"tm_renew,omitempty"`
	Days    int    `json:"days,omitempty"`
	Hours   int    `json:"hours,omitempty"`
}

// V1DataNumbersIDPut is
// v1 data type request struct for
// /v1/numbers/<id> PUT
type V1DataNumbersIDPut struct {
	CallFlowID    uuid.UUID `json:"call_flow_id,omitempty"`
	MessageFlowID uuid.UUID `json:"message_flow_id,omitempty"`
	Name          string    `json:"name,omitempty"`
	Detail        string    `json:"detail,omitempty"`
}

// V1DataNumbersIDFlowIDPut is
// v1 data type request struct for
// /v1/numbers/<id>/flow_id PUT
type V1DataNumbersIDFlowIDPut struct {
	CallFlowID    uuid.UUID `json:"call_flow_id"`
	MessageFlowID uuid.UUID `json:"message_flow_id"`
}

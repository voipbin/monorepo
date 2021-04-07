package request

import "github.com/gofrs/uuid"

// V1DataNumbersPost is
// v1 data type request struct for
// /v1/numbers POST
type V1DataNumbersPost struct {
	UserID uint64 `json:"user_id"`
	Number string `json:"number"`
}

// V1DataNumbersIDPut is
// v1 data type request struct for
// /v1/numbers/<id> PUT
type V1DataNumbersIDPut struct {
	FlowID uuid.UUID `json:"flow_id"`
}

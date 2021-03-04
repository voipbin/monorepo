package request

import "github.com/gofrs/uuid"

// NMV1DataNumbersPost is
// v1 data type request struct for
// /v1/numbers POST to number-manager
type NMV1DataNumbersPost struct {
	UserID uint64 `json:"user_id"`
	Number string `json:"number"`
}

// NMV1DataNumbersIDPut is
// v1 data type request struct for
// /v1/numbers/<id> PUT to number-manager
type NMV1DataNumbersIDPut struct {
	FlowID uuid.UUID `json:"flow_id"`
}

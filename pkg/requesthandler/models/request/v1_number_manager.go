package request

// NMV1DataNumbersPost is
// v1 data type request struct for
// /v1/numbers POST to number-manager
type NMV1DataNumbersPost struct {
	UserID uint64 `json:"user_id"`
	Number string `json:"number"`
}

package request

// NMV1DataOrderNumbersPost is
// v1 data type request struct for
// /v1/order_numbers POST to number-manager
type NMV1DataOrderNumbersPost struct {
	UserID uint64 `json:"user_id"`
	Number string `json:"number"`
}

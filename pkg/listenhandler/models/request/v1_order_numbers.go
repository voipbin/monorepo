package request

// V1DataNumbersPost is
// v1 data type request struct for
// /v1/order_numbers POST
type V1DataNumbersPost struct {
	UserID uint64 `json:"user_id"`
	Number string `json:"number"`
}

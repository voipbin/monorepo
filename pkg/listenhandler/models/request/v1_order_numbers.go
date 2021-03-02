package request

// V1DataOrderNumbersPost is
// v1 data type request struct for
// /v1/order_numbers POST
type V1DataOrderNumbersPost struct {
	UserID uint64 `json:"user_id"`
	Number string `json:"number"`
}

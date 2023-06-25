package response

// V1ResponseCustomersIDIsValidBalancePost is
// v1 response type for
// /v1/customers/<customer-id>/is_valid_balance POST
type V1ResponseCustomersIDIsValidBalancePost struct {
	Valid bool `json:"valid"`
}

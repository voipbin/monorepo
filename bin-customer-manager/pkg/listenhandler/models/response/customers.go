package response

// V1ResponseCustomersIDIsValidBalancePost is
// v1 response type for
// /v1/customers/<customer-id>/is_valid_balance POST
type V1ResponseCustomersIDIsValidBalancePost struct {
	Valid bool `json:"valid"`
}

// V1ResponseCustomersIDIsValidResourceLimitPost is
// v1 response type for
// /v1/customers/<customer-id>/is_valid_resource_limit POST
type V1ResponseCustomersIDIsValidResourceLimitPost struct {
	Valid bool `json:"valid"`
}

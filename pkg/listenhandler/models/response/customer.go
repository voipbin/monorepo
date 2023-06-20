package response

// V1ResponseCustomerIDIsValidBalance is
// v1 response type for
// /v1/customers/<customer-id>/is_valid_balance POST
type V1ResponseCustomerIDIsValidBalance struct {
	Valid bool `json:"valid"`
}

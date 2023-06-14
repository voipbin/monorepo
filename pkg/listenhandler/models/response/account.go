package response

// V1ResponseAccountsIDIsValidBalance is
// v1 response type for
// /v1/accounts/<account-id>/is_valid_balance POST
type V1ResponseAccountsIDIsValidBalance struct {
	Valid bool `json:"valid"`
}

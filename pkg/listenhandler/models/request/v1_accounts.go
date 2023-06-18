package request

// V1DataAccountsGET is rquest param define for GET /accounts
type V1DataAccountsGET struct {
	Pagination
}

// V1DataAccountsIDBalanceAddForcePOST is rquest param define for POST /accounts/<account-id>/balance_add
type V1DataAccountsIDBalanceAddForcePOST struct {
	Balance float32 `json:"balance"`
}

// V1DataAccountsIDBalanceSubtractForcePOST is rquest param define for POST /accounts/<account-id>/balance_subtract
type V1DataAccountsIDBalanceSubtractForcePOST struct {
	Balance float32 `json:"balance"`
}

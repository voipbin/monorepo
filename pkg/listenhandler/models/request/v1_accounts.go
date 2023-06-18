package request

// V1DataAccountsGET is rquest param define for GET /accounts
type V1DataAccountsGET struct {
	Pagination
}

// V1DataAccountsIDBalanceAddPOST is rquest param define for POST /accounts/<account-id>/balance_add
type V1DataAccountsIDBalanceAddPOST struct {
	Balance float32 `json:"balance"`
}

// V1DataAccountsIDBalanceSubtractPOST is rquest param define for POST /accounts/<account-id>/balance_subtract
type V1DataAccountsIDBalanceSubtractPOST struct {
	Balance float32 `json:"balance"`
}

package request

// BodyBillingAccountsPOST is rquest body define for
// POST /v1.0/billing_accounts
type BodyBillingAccountsPOST struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// ParamBillingAccountsGET is rquest param define for
// GET /v1.0/billing_accounts
type ParamBillingAccountsGET struct {
	Pagination
}

// BodyBillingAccountsIDBalanceAddForcePOST is rquest body define for
// POST /v1.0/billing_accounts/<billing_account_id>/balance_add_force
type BodyBillingAccountsIDBalanceAddForcePOST struct {
	Balance float32 `json:"balance"`
}

// BodyBillingAccountsIDBalanceSubtractForcePOST is rquest body define for
// POST /v1.0/billing_accounts/<billing_account_id>/balance_subtract_force
type BodyBillingAccountsIDBalanceSubtractForcePOST struct {
	Balance float32 `json:"balance"`
}

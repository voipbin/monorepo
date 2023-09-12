package request

import (
	bmaccount "gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
)

// BodyBillingAccountsPOST is rquest body define for
// POST /v1.0/billing_accounts
type BodyBillingAccountsPOST struct {
	Name          string                  `json:"name,omitempty"`
	Detail        string                  `json:"detail,omitempty"`
	PaymentType   bmaccount.PaymentType   `json:"payment_type,omitempty"`
	PaymentMethod bmaccount.PaymentMethod `json:"payment_method,omitempty"`
}

// ParamBillingAccountsGET is rquest param define for
// GET /v1.0/billing_accounts
type ParamBillingAccountsGET struct {
	Pagination
}

// BodyBillingAccountsIDPUT is rquest body define for
// PUT /v1.0/billing_accounts/<billing_account_id>
type BodyBillingAccountsIDPUT struct {
	Name   string `json:"name,omitempty"`
	Detail string `json:"detail,omitempty"`
}

// BodyBillingAccountsIDPaymentInfoPUT is rquest body define for
// PUT /v1.0/billing_accounts/<billing_account_id>/payment_info
type BodyBillingAccountsIDPaymentInfoPUT struct {
	PaymentType   bmaccount.PaymentType   `json:"payment_type,omitempty"`
	PaymentMethod bmaccount.PaymentMethod `json:"payment_method,omitempty"`
}

// BodyBillingAccountsIDBalanceAddForcePOST is rquest body define for
// POST /v1.0/billing_accounts/<billing_account_id>/balance_add_force
type BodyBillingAccountsIDBalanceAddForcePOST struct {
	Balance float32 `json:"balance,omitempty"`
}

// BodyBillingAccountsIDBalanceSubtractForcePOST is rquest body define for
// POST /v1.0/billing_accounts/<billing_account_id>/balance_subtract_force
type BodyBillingAccountsIDBalanceSubtractForcePOST struct {
	Balance float32 `json:"balance,omitempty"`
}

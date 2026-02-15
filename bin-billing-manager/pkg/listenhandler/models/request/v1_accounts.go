package request

import (
	"monorepo/bin-billing-manager/models/account"
)

// V1DataAccountsIDPUT is rquest param define for PUT /accounts/<account-id>
type V1DataAccountsIDPUT struct {
	Name   string `json:"name"`
	Detail string `json:"detail"`
}

// V1DataAccountsIDBalanceAddForcePOST is rquest param define for POST /accounts/<account-id>/balance_add
type V1DataAccountsIDBalanceAddForcePOST struct {
	Balance int64 `json:"balance"`
}

// V1DataAccountsIDBalanceSubtractForcePOST is rquest param define for POST /accounts/<account-id>/balance_subtract
type V1DataAccountsIDBalanceSubtractForcePOST struct {
	Balance int64 `json:"balance"`
}

// V1DataAccountsIDIsValidBalancePOST is rquest param define for POST /accounts/<account-id>/is_valid_balance
type V1DataAccountsIDIsValidBalancePOST struct {
	BillingType string `json:"billing_type"`
	Country     string `json:"country"`
	Count       int    `json:"count"`
}

// V1DataAccountsIDIsValidResourceLimitPOST is rquest param define for POST /accounts/<account-id>/is_valid_resource_limit
type V1DataAccountsIDIsValidResourceLimitPOST struct {
	ResourceType string `json:"resource_type"`
}

// V1DataAccountsIsValidBalanceByCustomerIDPOST is request param define for POST /accounts/is_valid_balance_by_customer_id
type V1DataAccountsIsValidBalanceByCustomerIDPOST struct {
	CustomerID  string `json:"customer_id"`
	BillingType string `json:"billing_type"`
	Country     string `json:"country"`
	Count       int    `json:"count"`
}

// V1DataAccountsIsValidResourceLimitByCustomerIDPOST is request param define for POST /accounts/is_valid_resource_limit_by_customer_id
type V1DataAccountsIsValidResourceLimitByCustomerIDPOST struct {
	CustomerID   string `json:"customer_id"`
	ResourceType string `json:"resource_type"`
}

// V1DataAccountsIDPaymentInfoPUT is rquest param define for POST /accounts/<account-id>/payment_info
type V1DataAccountsIDPaymentInfoPUT struct {
	PaymentType   account.PaymentType   `json:"payment_type"`
	PaymentMethod account.PaymentMethod `json:"payment_method"`
}

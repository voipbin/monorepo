package response

import (
	bmaccount "monorepo/bin-billing-manager/models/account"
)

// BodyBillingAccountsGET is response body define for
// GET /v1.0/billing_accounts
type BodyBillingAccountsGET struct {
	Result []*bmaccount.WebhookMessage `json:"result"`
	Pagination
}

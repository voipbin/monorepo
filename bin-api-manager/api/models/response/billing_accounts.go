package response

import (
	bmaccount "gitlab.com/voipbin/bin-manager/billing-manager.git/models/account"
)

// BodyBillingAccountsGET is response body define for
// GET /v1.0/billing_accounts
type BodyBillingAccountsGET struct {
	Result []*bmaccount.WebhookMessage `json:"result"`
	Pagination
}

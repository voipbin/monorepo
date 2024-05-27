package response

import (
	smaccount "monorepo/bin-storage-manager/models/account"
)

// BodyStorageAccountsGET is rquest body define for
// GET /v1.0/storage_accounts
type BodyStorageAccountsGET struct {
	Result []*smaccount.WebhookMessage `json:"result"`
	Pagination
}

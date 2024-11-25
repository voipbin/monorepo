package response

import (
	csaccesskey "monorepo/bin-customer-manager/models/accesskey"
)

// BodyAccesskeysGET is response body define for
// GET /v1.0/accesskeys
type BodyAccesskeysGET struct {
	Result []*csaccesskey.WebhookMessage `json:"result"`
	Pagination
}

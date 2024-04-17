package response

import (
	nmnumber "monorepo/bin-number-manager/models/number"
)

// BodyNumbersGET is rquest body define for
// GET /v1.0/numbers
type BodyNumbersGET struct {
	Result []*nmnumber.WebhookMessage `json:"result"`
	Pagination
}

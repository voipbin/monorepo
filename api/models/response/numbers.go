package response

import (
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// BodyNumbersGET is rquest body define for
// GET /v1.0/numbers
type BodyNumbersGET struct {
	Result []*nmnumber.WebhookMessage `json:"result"`
	Pagination
}

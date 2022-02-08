package response

import (
	nmnumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/number"
)

// BodyNumbersGET is rquest body define for GET /numbers
type BodyNumbersGET struct {
	Result []*nmnumber.WebhookMessage `json:"result"`
	Pagination
}

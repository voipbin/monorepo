package response

import nmavailablenumber "gitlab.com/voipbin/bin-manager/number-manager.git/models/availablenumber"

// BodyAvailableNumbersGET is rquest body define for
// GET /v1.0/available_numbers
type BodyAvailableNumbersGET struct {
	Result []*nmavailablenumber.WebhookMessage `json:"result"`
}

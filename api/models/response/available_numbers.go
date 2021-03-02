package response

import (
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/number"
)

// BodyAvailableNumbersGET is rquest body define for GET /available_numbers
type BodyAvailableNumbersGET struct {
	Result []*number.AvailableNumber `json:"result"`
}

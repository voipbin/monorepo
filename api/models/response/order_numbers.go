package response

import (
	"gitlab.com/voipbin/bin-manager/api-manager.git/models/number"
)

// BodyOrderNumbersGET is rquest body define for GET /order_numbers
type BodyOrderNumbersGET struct {
	Result []*number.Number `json:"result"`
	Pagination
}

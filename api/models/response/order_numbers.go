package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models/number"

// BodyNumbersGET is rquest body define for GET /numbers
type BodyNumbersGET struct {
	Result []*number.Number `json:"result"`
	Pagination
}

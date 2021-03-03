package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models"

// BodyOrderNumbersGET is rquest body define for GET /order_numbers
type BodyOrderNumbersGET struct {
	Result []*models.Number `json:"result"`
	Pagination
}

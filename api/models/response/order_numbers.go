package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models"

// BodyNumbersGET is rquest body define for GET /numbers
type BodyNumbersGET struct {
	Result []*models.Number `json:"result"`
	Pagination
}

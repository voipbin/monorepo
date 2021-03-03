package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models"

// BodyFlowsGET is rquest body define for GET /flows
type BodyFlowsGET struct {
	Result []*models.Flow `json:"result"`
	Pagination
}

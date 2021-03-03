package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models"

// BodyCallsGET is rquest body define for GET /calls
type BodyCallsGET struct {
	Result []*models.Call `json:"result"`
	Pagination
}

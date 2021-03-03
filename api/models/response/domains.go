package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models"

// BodyDomainsGET is rquest body define for GET /domains
type BodyDomainsGET struct {
	Result []*models.Domain `json:"result"`
	Pagination
}

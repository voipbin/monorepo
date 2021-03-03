package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models"

// BodyExtensionsGET is rquest body define for GET /extensions
type BodyExtensionsGET struct {
	Result []*models.Extension `json:"result"`
	Pagination
}

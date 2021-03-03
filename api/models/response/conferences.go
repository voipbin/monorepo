package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models"

// BodyConferencesGET is rquest body define for GET /calls
type BodyConferencesGET struct {
	Result []*models.Conference `json:"result"`
	Pagination
}

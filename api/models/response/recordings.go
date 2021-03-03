package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models"

// BodyRecordingsGET is rquest body define for GET /recordings
type BodyRecordingsGET struct {
	Result []*models.Recording `json:"result"`
	Pagination
}

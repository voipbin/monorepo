package response

import "gitlab.com/voipbin/bin-manager/api-manager.git/models/recording"

// BodyRecordingsGET is rquest body define for GET /recordings
type BodyRecordingsGET struct {
	Result []*recording.Recording `json:"result"`
	Pagination
}

package response

import (
	cmrecording "gitlab.com/voipbin/bin-manager/call-manager.git/models/recording"
)

// BodyRecordingsGET is rquest body define for
// GET /v1.0/recordings
type BodyRecordingsGET struct {
	Result []*cmrecording.WebhookMessage `json:"result"`
	Pagination
}

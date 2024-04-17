package response

import (
	cmrecording "monorepo/bin-call-manager/models/recording"
)

// BodyRecordingsGET is rquest body define for
// GET /v1.0/recordings
type BodyRecordingsGET struct {
	Result []*cmrecording.WebhookMessage `json:"result"`
	Pagination
}

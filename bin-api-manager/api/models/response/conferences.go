package response

import (
	cfconference "monorepo/bin-conference-manager/models/conference"
)

// BodyConferencesGET is rquest body define for
// GET /v1.0/calls
type BodyConferencesGET struct {
	Result []*cfconference.WebhookMessage `json:"result"`
	Pagination
}

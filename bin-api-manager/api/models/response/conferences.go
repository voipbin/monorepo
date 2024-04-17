package response

import (
	cfconference "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conference"
)

// BodyConferencesGET is rquest body define for
// GET /v1.0/calls
type BodyConferencesGET struct {
	Result []*cfconference.WebhookMessage `json:"result"`
	Pagination
}

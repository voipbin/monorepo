package response

import (
	cfconferencecall "gitlab.com/voipbin/bin-manager/conference-manager.git/models/conferencecall"
)

// BodyConferencecallsGET is rquest body define for
// GET /v1.0/conferencecalls
type BodyConferencecallsGET struct {
	Result []*cfconferencecall.WebhookMessage `json:"result"`
	Pagination
}

package response

import (
	cfconferencecall "monorepo/bin-conference-manager/models/conferencecall"
)

// BodyConferencecallsGET is rquest body define for
// GET /v1.0/conferencecalls
type BodyConferencecallsGET struct {
	Result []*cfconferencecall.WebhookMessage `json:"result"`
	Pagination
}
